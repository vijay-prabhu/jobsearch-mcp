package database

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/001_initial.sql
var initialMigration string

//go:embed migrations/002_add_archived.sql
var archivedMigration string

//go:embed migrations/003_performance_indexes.sql
var performanceIndexesMigration string

//go:embed migrations/004_learned_filters.sql
var learnedFiltersMigration string

//go:embed migrations/005_review_suggested.sql
var reviewSuggestedMigration string

// DB wraps the SQL database connection
type DB struct {
	*sql.DB
}

// Open opens or creates the database at the given path
func Open(path string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database with common settings
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON", path)
	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes
	sqlDB.SetMaxIdleConns(1)

	db := &DB{sqlDB}

	// Run migrations
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// MustOpen opens the database or panics
func MustOpen(path string) *DB {
	db, err := Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	return db
}

// migrate runs database migrations
func (db *DB) migrate() error {
	// Check if we need to run initial migration
	var tableCount int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name='conversations'
	`).Scan(&tableCount)
	if err != nil {
		return fmt.Errorf("failed to check migrations: %w", err)
	}

	if tableCount == 0 {
		// Run initial migration
		if _, err := db.Exec(initialMigration); err != nil {
			return fmt.Errorf("failed to run initial migration: %w", err)
		}
	}

	// Check if archived column exists (migration 002)
	var archivedExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('conversations')
		WHERE name='archived'
	`).Scan(&archivedExists)
	if err != nil {
		return fmt.Errorf("failed to check archived column: %w", err)
	}

	if archivedExists == 0 {
		// Run archived migration
		if _, err := db.Exec(archivedMigration); err != nil {
			return fmt.Errorf("failed to run archived migration: %w", err)
		}
	}

	// Check if performance indexes exist (migration 003)
	var perfIndexExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='index' AND name='idx_conversations_status_archived'
	`).Scan(&perfIndexExists)
	if err != nil {
		return fmt.Errorf("failed to check performance indexes: %w", err)
	}

	if perfIndexExists == 0 {
		// Run performance indexes migration
		if _, err := db.Exec(performanceIndexesMigration); err != nil {
			return fmt.Errorf("failed to run performance indexes migration: %w", err)
		}
	}

	// Check if learned_filters table exists (migration 004)
	var learnedFiltersExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name='learned_filters'
	`).Scan(&learnedFiltersExists)
	if err != nil {
		return fmt.Errorf("failed to check learned_filters table: %w", err)
	}

	if learnedFiltersExists == 0 {
		// Run learned filters migration
		if _, err := db.Exec(learnedFiltersMigration); err != nil {
			return fmt.Errorf("failed to run learned filters migration: %w", err)
		}
	} else {
		// Check if we need to upgrade the learned_filters table schema
		var fpCountExists int
		err = db.QueryRow(`
			SELECT COUNT(*) FROM pragma_table_info('learned_filters')
			WHERE name='false_positive_count'
		`).Scan(&fpCountExists)
		if err != nil {
			return fmt.Errorf("failed to check false_positive_count column: %w", err)
		}

		if fpCountExists == 0 {
			// Need to recreate table with new schema
			// SQLite doesn't support adding columns with defaults easily, so recreate
			_, err = db.Exec(`
				DROP TABLE IF EXISTS learned_filters;
				DROP TABLE IF EXISTS classification_metrics;
			`)
			if err != nil {
				return fmt.Errorf("failed to drop old learned_filters table: %w", err)
			}
			if _, err := db.Exec(learnedFiltersMigration); err != nil {
				return fmt.Errorf("failed to recreate learned filters table: %w", err)
			}
		}
	}

	// Check if review_suggested column exists (migration 005)
	var reviewSuggestedExists int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('conversations')
		WHERE name='review_suggested'
	`).Scan(&reviewSuggestedExists)
	if err != nil {
		return fmt.Errorf("failed to check review_suggested column: %w", err)
	}

	if reviewSuggestedExists == 0 {
		// Run review_suggested migration
		if _, err := db.Exec(reviewSuggestedMigration); err != nil {
			return fmt.Errorf("failed to run review_suggested migration: %w", err)
		}
	}

	return nil
}

// Transaction runs a function in a transaction
func (db *DB) Transaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback() // Ignore rollback error since we're returning the original error
		return err
	}

	return tx.Commit()
}

// Health checks database connectivity
func (db *DB) Health(ctx context.Context) error {
	return db.PingContext(ctx)
}

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

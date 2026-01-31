package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "jobsearch-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to open database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestOpen(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if db == nil {
		t.Fatal("expected non-nil database")
	}

	// Verify tables exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='conversations'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query tables: %v", err)
	}
	if count != 1 {
		t.Errorf("expected conversations table to exist")
	}
}

func TestConversationCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Create
	recruiterEmail := "recruiter@company.com"
	recruiterName := "Jane Recruiter"
	conv := &Conversation{
		Company:        "TestCorp",
		RecruiterEmail: &recruiterEmail,
		RecruiterName:  &recruiterName,
		Direction:      DirectionInbound,
		Status:         StatusActive,
		LastActivityAt: time.Now(),
		EmailCount:     0,
	}

	err := db.CreateConversation(ctx, conv)
	if err != nil {
		t.Fatalf("CreateConversation failed: %v", err)
	}
	if conv.ID == "" {
		t.Error("expected ID to be set after create")
	}

	// Read
	fetched, err := db.GetConversation(ctx, conv.ID)
	if err != nil {
		t.Fatalf("GetConversation failed: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected conversation to be found")
	}
	if fetched.Company != "TestCorp" {
		t.Errorf("expected Company=TestCorp, got %s", fetched.Company)
	}

	// Update
	conv.Status = StatusWaitingOnMe
	err = db.UpdateConversation(ctx, conv)
	if err != nil {
		t.Fatalf("UpdateConversation failed: %v", err)
	}

	fetched, _ = db.GetConversation(ctx, conv.ID)
	if fetched.Status != StatusWaitingOnMe {
		t.Errorf("expected Status=waiting_on_me, got %s", fetched.Status)
	}

	// List
	convs, err := db.ListConversations(ctx, ListOptions{})
	if err != nil {
		t.Fatalf("ListConversations failed: %v", err)
	}
	if len(convs) != 1 {
		t.Errorf("expected 1 conversation, got %d", len(convs))
	}
}

func TestGetConversationByCompany(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	conv := &Conversation{
		Company:        "Stripe",
		Direction:      DirectionInbound,
		Status:         StatusActive,
		LastActivityAt: time.Now(),
	}
	db.CreateConversation(ctx, conv)

	// Test case-insensitive search
	found, err := db.GetConversationByCompany(ctx, "stripe")
	if err != nil {
		t.Fatalf("GetConversationByCompany failed: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find conversation")
	}
	if found.Company != "Stripe" {
		t.Errorf("expected Company=Stripe, got %s", found.Company)
	}

	// Test not found
	notFound, _ := db.GetConversationByCompany(ctx, "Google")
	if notFound != nil {
		t.Error("expected nil for non-existent company")
	}
}

func TestEmailCRUD(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Create conversation first
	conv := &Conversation{
		Company:        "TestCorp",
		Direction:      DirectionInbound,
		Status:         StatusActive,
		LastActivityAt: time.Now(),
	}
	db.CreateConversation(ctx, conv)

	// Create email
	subject := "Test Subject"
	email := &Email{
		ConversationID: conv.ID,
		GmailID:        "gmail-123",
		ThreadID:       "thread-456",
		Subject:        &subject,
		FromAddress:    "recruiter@company.com",
		Date:           time.Now(),
		Direction:      DirectionInbound,
	}

	err := db.CreateEmail(ctx, email)
	if err != nil {
		t.Fatalf("CreateEmail failed: %v", err)
	}

	// Get by Gmail ID
	fetched, err := db.GetEmailByGmailID(ctx, "gmail-123")
	if err != nil {
		t.Fatalf("GetEmailByGmailID failed: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected email to be found")
	}
	if *fetched.Subject != "Test Subject" {
		t.Errorf("expected Subject='Test Subject', got %s", *fetched.Subject)
	}

	// List for conversation
	emails, err := db.ListEmailsForConversation(ctx, conv.ID)
	if err != nil {
		t.Fatalf("ListEmailsForConversation failed: %v", err)
	}
	if len(emails) != 1 {
		t.Errorf("expected 1 email, got %d", len(emails))
	}
}

func TestListConversationsWithFilters(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Create test conversations
	convs := []Conversation{
		{Company: "Stripe", Direction: DirectionInbound, Status: StatusWaitingOnMe, LastActivityAt: time.Now()},
		{Company: "Google", Direction: DirectionInbound, Status: StatusWaitingOnThem, LastActivityAt: time.Now()},
		{Company: "Meta", Direction: DirectionOutbound, Status: StatusStale, LastActivityAt: time.Now().Add(-30 * 24 * time.Hour)},
	}
	for i := range convs {
		db.CreateConversation(ctx, &convs[i])
	}

	// Test status filter
	status := StatusWaitingOnMe
	results, _ := db.ListConversations(ctx, ListOptions{Status: &status})
	if len(results) != 1 {
		t.Errorf("expected 1 waiting_on_me conversation, got %d", len(results))
	}

	// Test limit
	results, _ = db.ListConversations(ctx, ListOptions{Limit: 2})
	if len(results) != 2 {
		t.Errorf("expected 2 conversations with limit, got %d", len(results))
	}
}

func TestSyncState(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Get initial state
	state, err := db.GetSyncState(ctx)
	if err != nil {
		t.Fatalf("GetSyncState failed: %v", err)
	}
	if state.EmailsProcessed != 0 {
		t.Errorf("expected EmailsProcessed=0, got %d", state.EmailsProcessed)
	}

	// Update state
	now := time.Now()
	historyID := "12345"
	state.LastSyncAt = &now
	state.LastHistoryID = &historyID
	state.EmailsProcessed = 100

	err = db.UpdateSyncState(ctx, state)
	if err != nil {
		t.Fatalf("UpdateSyncState failed: %v", err)
	}

	// Verify
	updated, _ := db.GetSyncState(ctx)
	if updated.EmailsProcessed != 100 {
		t.Errorf("expected EmailsProcessed=100, got %d", updated.EmailsProcessed)
	}
}

func TestSearch(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	recruiter := "Jane Smith"
	conv := &Conversation{
		Company:        "Stripe",
		RecruiterName:  &recruiter,
		Direction:      DirectionInbound,
		Status:         StatusActive,
		LastActivityAt: time.Now(),
	}
	db.CreateConversation(ctx, conv)

	// Search by company
	results, _ := db.Search(ctx, "stripe")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'stripe', got %d", len(results))
	}

	// Search by recruiter
	results, _ = db.Search(ctx, "jane")
	if len(results) != 1 {
		t.Errorf("expected 1 result for 'jane', got %d", len(results))
	}

	// Search no results
	results, _ = db.Search(ctx, "nonexistent")
	if len(results) != 0 {
		t.Errorf("expected 0 results for 'nonexistent', got %d", len(results))
	}
}

func TestGetStats(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()

	// Create test data
	convs := []Conversation{
		{Company: "A", Direction: DirectionInbound, Status: StatusWaitingOnMe, LastActivityAt: time.Now()},
		{Company: "B", Direction: DirectionInbound, Status: StatusWaitingOnThem, LastActivityAt: time.Now()},
		{Company: "C", Direction: DirectionInbound, Status: StatusStale, LastActivityAt: time.Now()},
	}
	for i := range convs {
		db.CreateConversation(ctx, &convs[i])
	}

	stats, err := db.GetStats(ctx, nil)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalConversations != 3 {
		t.Errorf("expected TotalConversations=3, got %d", stats.TotalConversations)
	}
	if stats.WaitingOnMe != 1 {
		t.Errorf("expected WaitingOnMe=1, got %d", stats.WaitingOnMe)
	}
	if stats.WaitingOnThem != 1 {
		t.Errorf("expected WaitingOnThem=1, got %d", stats.WaitingOnThem)
	}
	if stats.Stale != 1 {
		t.Errorf("expected Stale=1, got %d", stats.Stale)
	}
}

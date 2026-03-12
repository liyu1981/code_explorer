package db

import (
	"testing"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

func TestResearchStore(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	// Need a codebase
	cb, _ := store.GetOrCreateCodebase("/test", "test", "local")

	// Test Save Session
	sessionID, _ := gonanoid.New()
	session := &ResearchSession{
		ID:         sessionID,
		CodebaseID: cb.ID,
		Title:      "Test Session",
		CreatedAt:  time.Now().UnixMilli(),
	}
	if err := store.SaveResearchSession(session); err != nil {
		t.Fatalf("save session: %v", err)
	}

	// Test Get Session
	got, err := store.GetResearchSession(sessionID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if got == nil || got.Title != "Test Session" {
		t.Errorf("expected title 'Test Session', got %v", got)
	}

	// Test Save Report Chunk
	turnID := "turn-1"
	if err := store.SaveResearchReportChunk(sessionID, turnID, "part 1 "); err != nil {
		t.Fatalf("save chunk 1: %v", err)
	}
	if err := store.SaveResearchReportChunk(sessionID, turnID, "part 2"); err != nil {
		t.Fatalf("save chunk 2: %v", err)
	}

	// Test Get Reports
	reports, err := store.GetResearchReportsBySession(sessionID)
	if err != nil {
		t.Fatalf("get reports: %v", err)
	}
	if len(reports) != 1 || reports[0].StreamData != "part 1 part 2" {
		t.Errorf("expected data 'part 1 part 2', got %v", reports)
	}

	// Test Saved Report
	saved := &SavedReport{
		SessionID:    sessionID,
		CodebaseID:   cb.ID,
		Title:        "Saved Title",
		Query:        "query",
		Content:      "content",
		CodebaseName: "test",
		CodebasePath: "/test",
	}
	if err := store.SaveSavedReport(saved); err != nil {
		t.Fatalf("save saved report: %v", err)
	}
	if saved.ID == "" {
		t.Error("expected ID to be set")
	}

	// Test List Saved Reports
	list, total, err := store.ListSavedReports(1, 10, "")
	if err != nil {
		t.Fatalf("list saved reports: %v", err)
	}
	if total != 1 || list[0].Title != "Saved Title" {
		t.Errorf("expected 1 saved report, got %d (total %d)", len(list), total)
	}

	// Test Delete Session (should cascade delete reports)
	if err := store.DeleteResearchSession(sessionID); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	got, _ = store.GetResearchSession(sessionID)
	if got != nil {
		t.Error("expected session to be deleted")
	}
	reports, _ = store.GetResearchReportsBySession(sessionID)
	if len(reports) != 0 {
		t.Error("expected reports to be deleted")
	}
}

package db

import (
	"context"
	"testing"
)

func TestResearchStore(t *testing.T) {
	store, cleanup := SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	cb, _ := store.GetOrCreateCodebase(ctx, "/test", "test", "local")

	session := &ResearchSession{
		ID:         "sess1",
		CodebaseID: cb.ID,
		Title:      "Test Session",
		State:      "{}",
		CreatedAt:  123,
	}

	// Test Save Session
	if err := store.SaveResearchSession(ctx, session); err != nil {
		t.Fatalf("save session: %v", err)
	}

	// Test Get Session
	got, err := store.GetResearchSession(ctx, "sess1")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if got.Title != "Test Session" {
		t.Errorf("expected title Test Session, got %s", got.Title)
	}

	// Test Save Report Chunk
	if err := store.SaveResearchReportChunk(ctx, "sess1", "turn1", "hello"); err != nil {
		t.Fatalf("save report chunk: %v", err)
	}
	if err := store.SaveResearchReportChunk(ctx, "sess1", "turn1", " world"); err != nil {
		t.Fatalf("save report chunk append: %v", err)
	}

	// Test Get Reports
	reports, err := store.GetResearchReportsBySession(ctx, "sess1")
	if err != nil {
		t.Fatalf("get reports: %v", err)
	}
	if len(reports) != 1 || reports[0].StreamData != "hello world" {
		t.Errorf("expected hello world, got %s", reports[0].StreamData)
	}

	// Test Paginated
	sessions, total, err := store.GetResearchSessionsPaginated(ctx, cb.ID, 1, 10)
	if err != nil {
		t.Fatalf("paginated: %v", err)
	}
	if total != 1 || len(sessions) != 1 {
		t.Errorf("expected 1 session, got %d", total)
	}

	// Test Save Saved Report
	sr := &SavedReport{
		ID:         "sr1",
		SessionID:  "sess1",
		CodebaseID: cb.ID,
		Title:      "Saved",
		Content:    "Content",
	}
	if err := store.SaveSavedReport(ctx, sr); err != nil {
		t.Fatalf("save saved report: %v", err)
	}

	// Test Get Saved Report
	gsr, err := store.GetSavedReport(ctx, "sr1")
	if err != nil {
		t.Fatalf("get saved report: %v", err)
	}
	if gsr.Title != "Saved" {
		t.Errorf("expected Saved, got %s", gsr.Title)
	}

	// Test List Saved Reports
	lrs, total, err := store.ListSavedReports(ctx, 1, 10, "")
	if err != nil {
		t.Fatalf("list saved reports: %v", err)
	}
	if total != 1 || len(lrs) != 1 {
		t.Errorf("expected 1 saved report, got %d", total)
	}

	// Test Delete Session
	if err := store.DeleteResearchSession(ctx, "sess1"); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	got, _ = store.GetResearchSession(ctx, "sess1")
	if got != nil {
		t.Error("expected session to be deleted")
	}
}

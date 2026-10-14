package syncer_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"trello_analytics_back/internal/importer"
	"trello_analytics_back/internal/storage/postgres"
	"trello_analytics_back/internal/syncer"
)

// stubStatusStore drives Status without Postgres (avoids truncate races with importer tests).
type stubStatusStore struct {
	lastSuccess *time.Time
	hasCursor   bool
	latest      *postgres.SyncRun
}

func (s *stubStatusStore) InsertSyncRun(context.Context, time.Time) (string, error) {
	return "", nil
}
func (s *stubStatusStore) FinishSyncRun(context.Context, string, string, postgres.SyncRunCounts, *string) error {
	return nil
}
func (s *stubStatusStore) ReclaimOrphanSyncRuns(context.Context) (int64, error) { return 0, nil }
func (s *stubStatusStore) GetSyncRun(context.Context, string) (*postgres.SyncRun, error) {
	return nil, nil
}
func (s *stubStatusStore) GetRunningSyncRun(context.Context) (*postgres.SyncRun, error) {
	return nil, nil
}
func (s *stubStatusStore) GetLatestSyncRun(context.Context) (*postgres.SyncRun, error) {
	return s.latest, nil
}
func (s *stubStatusStore) GetLastSuccessFinishedAt(context.Context) (*time.Time, error) {
	return s.lastSuccess, nil
}
func (s *stubStatusStore) ActionDateRange(context.Context) (*time.Time, *time.Time, error) {
	return nil, nil, nil
}
func (s *stubStatusStore) HasSyncCursor(_ context.Context, resourceType string) (bool, error) {
	if resourceType != postgres.ResourceBoardActions {
		return false, nil
	}
	return s.hasCursor, nil
}
func (s *stubStatusStore) CountCards(context.Context) (int64, error)       { return 0, nil }
func (s *stubStatusStore) CountLists(context.Context) (int64, error)       { return 0, nil }
func (s *stubStatusStore) CountMembers(context.Context) (int64, error)     { return 0, nil }
func (s *stubStatusStore) CountLabels(context.Context) (int64, error)      { return 0, nil }
func (s *stubStatusStore) CountCardActions(context.Context) (int64, error) { return 0, nil }

type noopImporter struct{}

func (noopImporter) ImportBoard(context.Context, importer.ImportBoardOptions) (importer.ImportBoardResult, error) {
	return importer.ImportBoardResult{}, nil
}

func TestStatus_EmptyBoardCompleteAfterSuccessAndCursor(t *testing.T) {
	successAt := time.Date(2024, 7, 1, 12, 0, 0, 0, time.UTC)
	store := &stubStatusStore{
		lastSuccess: &successAt,
		hasCursor:   true, // empty-board ImportBoardActions creates sync_cursors row
		latest: &postgres.SyncRun{
			ID:        "run-1",
			Status:    postgres.SyncRunSuccess,
			StartedAt: successAt.Add(-time.Minute),
			FinishedAt: func() *time.Time {
				t := successAt
				return &t
			}(),
		},
	}
	orch := syncer.NewOrchestrator(store, noopImporter{}, syncer.Config{BoardID: "board-empty"})
	st, err := orch.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.PrimaryImportIncomplete {
		t.Fatal("expected primary_import_incomplete=false after success + board_actions cursor (empty board)")
	}
	for _, w := range st.Warnings {
		if strings.Contains(w, "no head catch-up") || strings.Contains(w, "may lag live Trello") {
			t.Fatalf("unexpected obsolete head catch-up lag warning: %q", w)
		}
	}
	if len(st.Warnings) < 1 {
		t.Fatalf("expected default depth/card-feed warnings, got %#v", st.Warnings)
	}
}

func TestStatus_IncompleteWithoutCursor(t *testing.T) {
	successAt := time.Date(2024, 7, 1, 12, 0, 0, 0, time.UTC)
	store := &stubStatusStore{lastSuccess: &successAt, hasCursor: false}
	orch := syncer.NewOrchestrator(store, noopImporter{}, syncer.Config{BoardID: "board-x"})
	st, err := orch.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !st.PrimaryImportIncomplete {
		t.Fatal("expected primary_import_incomplete=true when board_actions cursor missing")
	}
}

package handlers

import (
	"context"
	"fmt"
	"net/http"

	"trello_analytics_back/internal/storage/postgres"
)

// resolveBoardID returns the internal board UUID for the configured Trello board.
func resolveBoardID(ctx context.Context, store *postgres.Store, trelloBoardID string) (string, error) {
	if trelloBoardID == "" {
		return "", fmt.Errorf("board not configured")
	}
	b, err := store.GetBoardByTrelloID(ctx, trelloBoardID)
	if err != nil {
		return "", err
	}
	if b == nil {
		return "", fmt.Errorf("board not synced")
	}
	return b.ID, nil
}

func writeBoardResolveError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch msg {
	case "board not configured":
		writeError(w, http.StatusServiceUnavailable, "board_not_configured", "TRELLO_BOARD_ID is not configured")
	case "board not synced":
		writeError(w, http.StatusServiceUnavailable, "board_not_synced", "board has not been imported yet; run sync first")
	default:
		WriteInternal(w)
	}
}

package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/codesummer"
	"github.com/liyu1981/code_explorer/pkg/db"
)

func HandleCodesummerTask(
	ctx context.Context,
	index *codemogger.CodeIndex,
	task *db.Task,
	updateProgress func(progress int, message string),
) error {
	var payload struct {
		CodebaseID string `json:"codebaseId"`
	}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	updateProgress(10, "Starting codesummer job...")

	store := index.GetStore()
	err := codesummer.Summary(ctx, store, payload.CodebaseID)
	if err != nil {
		return fmt.Errorf("codesummer failed: %w", err)
	}

	updateProgress(100, "Codesummer job completed")
	return nil
}

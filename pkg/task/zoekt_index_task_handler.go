package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/db"
	"github.com/liyu1981/code_explorer/pkg/zoekt"
	zkindex "github.com/liyu1981/code_explorer/pkg/zoekt/index"
)

func HandleZoektIndexTask(ctx context.Context, zIdx *zoekt.ZoektIndex, task *db.Task, updateProgress func(progress int, message string)) error {
	var payload struct {
		Dir   string   `json:"dir"`
		Langs []string `json:"langs"`
	}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	opts := &zkindex.IndexOptions{
		Languages: payload.Langs,
		Progress: func(current, total int, stage string) {
			progress := 0
			if total > 0 {
				progress = (current * 100) / total
			}
			switch stage {
			case "scan":
				progress = progress / 10
			case "check":
				progress = 10 + (progress / 10)
			case "index":
				progress = 20 + (progress * 80 / 100)
			}
			updateProgress(progress, fmt.Sprintf("Stage: %s (%d/%d)", stage, current, total))
		},
	}

	_, err := zIdx.Index(ctx, payload.Dir, opts)
	return err
}

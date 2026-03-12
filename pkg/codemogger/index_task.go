package codemogger

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/liyu1981/code_explorer/pkg/db"
)

func (c *CodeIndex) HandleIndexTask(ctx context.Context, task *db.Task, updateProgress func(progress int, message string)) error {
	var payload struct {
		Dir   string   `json:"dir"`
		Langs []string `json:"langs"`
	}
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	opts := &IndexOptions{
		Languages: payload.Langs,
		Progress: func(current, total int, stage string) {
			progress := 0
			if total > 0 {
				progress = (current * 100) / total
			}
			// Map stages to progress ranges
			switch stage {
			case "scan":
				progress = progress / 10 // 0-10%
			case "check":
				progress = 10 + (progress / 10) // 10-20%
			case "chunk":
				progress = 20 + (progress / 5) // 20-40%
			case "embed":
				progress = 40 + (progress * 60 / 100) // 40-100%
			}
			updateProgress(progress, fmt.Sprintf("Stage: %s (%d/%d)", stage, current, total))
		},
	}

	_, err := c.Index(payload.Dir, opts)
	return err
}

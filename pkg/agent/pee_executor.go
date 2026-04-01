package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/liyu1981/code_explorer/pkg/tools"
	"github.com/rs/zerolog/log"
)

type PEEExecutor struct {
	toolRegistry *tools.ToolRegistry
	maxWorkers   int
}

func NewPEEExecutor(toolRegistry *tools.ToolRegistry, maxWorkers int) *PEEExecutor {
	return &PEEExecutor{
		toolRegistry: toolRegistry,
		maxWorkers:   maxWorkers,
	}
}

func (e *PEEExecutor) Execute(ctx context.Context, d *DAG) error {
	sem := make(chan struct{}, e.maxWorkers)
	results := make(chan string, len(d.tasks))
	var wg sync.WaitGroup

	running := 0

	for !d.IsDone() {
		for _, t := range d.ReadyTasks() {
			select {
			case sem <- struct{}{}:
			default:
				continue
			}

			d.SetStatus(t.ID, StatusRunning)
			t.StartedAt = time.Now()
			wg.Add(1)
			running++

			go func(task *Task) {
				defer wg.Done()
				defer func() {
					<-sem
					results <- task.ID
				}()

				tool, ok := e.toolRegistry.Get(task.Tool)
				if !ok {
					log.Error().Str("tool", task.Tool).Msg("unknown tool")
					d.SetResult(task.ID, nil, fmt.Errorf("unknown tool: %s", task.Tool))
					d.SetStatus(task.ID, StatusFailed)
					e.skipDependents(d, task.ID)
					return
				}

				inputJSON, err := json.Marshal(task.Input)
				if err != nil {
					log.Error().Err(err).Str("task", task.ID).Msg("failed to marshal input")
					d.SetResult(task.ID, nil, fmt.Errorf("marshal input: %w", err))
					d.SetStatus(task.ID, StatusFailed)
					e.skipDependents(d, task.ID)
					return
				}

				output, err := tool.Execute(ctx, inputJSON, nil)
				d.SetResult(task.ID, output, err)
				if err != nil {
					log.Error().Err(err).Str("task", task.ID).Msg("tool execution failed")
					d.SetStatus(task.ID, StatusFailed)
					e.skipDependents(d, task.ID)
				} else {
					log.Info().Str("task", task.ID).Msg("task completed")
					d.SetStatus(t.ID, StatusDone)
				}
			}(t)
		}

		if running == 0 {
			break
		}

		<-results
		running--
	}

	wg.Wait()
	return nil
}

func (e *PEEExecutor) skipDependents(d *DAG, failedID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	queue := []string{failedID}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		for _, t := range d.tasks {
			for _, dep := range t.DependsOn {
				if dep == current && !visited[t.ID] {
					if t.Status == StatusPending || t.Status == StatusReady || t.Status == StatusRunning {
						t.Status = StatusSkipped
						queue = append(queue, t.ID)
					}
				}
			}
		}
	}
}

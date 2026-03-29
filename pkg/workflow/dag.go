package workflow

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type TaskStatus string

const (
	StatusPending TaskStatus = "pending"
	StatusReady   TaskStatus = "ready"
	StatusRunning TaskStatus = "running"
	StatusDone    TaskStatus = "done"
	StatusFailed  TaskStatus = "failed"
	StatusSkipped TaskStatus = "skipped"
)

type Task struct {
	ID          string
	Description string
	Tool        string
	Input       map[string]any
	DependsOn   []string

	Status     TaskStatus
	Output     any
	Err        error
	StartedAt  time.Time
	FinishedAt time.Time
}

type DAG struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

func NewDAG(tasks []*Task) (*DAG, error) {
	d := &DAG{tasks: make(map[string]*Task, len(tasks))}
	for _, t := range tasks {
		d.tasks[t.ID] = t
	}
	return d, d.validate()
}

func (d *DAG) validate() error {
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var dfs func(id string) error
	dfs = func(id string) error {
		if inStack[id] {
			return fmt.Errorf("cycle detected involving task: %s", id)
		}
		if visited[id] {
			return nil
		}

		visited[id] = true
		inStack[id] = true

		task, ok := d.tasks[id]
		if !ok {
			return fmt.Errorf("task not found: %s", id)
		}

		for _, dep := range task.DependsOn {
			if err := dfs(dep); err != nil {
				return err
			}
		}

		inStack[id] = false
		return nil
	}

	for id := range d.tasks {
		if err := dfs(id); err != nil {
			return err
		}
	}

	return nil
}

func (d *DAG) ReadyTasks() []*Task {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var ready []*Task
	for _, t := range d.tasks {
		if t.Status != StatusPending {
			continue
		}
		if d.depsAllDone(t) {
			ready = append(ready, t)
		}
	}
	return ready
}

func (d *DAG) depsAllDone(t *Task) bool {
	for _, dep := range t.DependsOn {
		if d.tasks[dep].Status != StatusDone {
			return false
		}
	}
	return true
}

func (d *DAG) SetStatus(id string, s TaskStatus) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.tasks[id].Status = s
}

func (d *DAG) SetResult(id string, output any, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	t := d.tasks[id]
	t.Output = output
	t.Err = err
	t.FinishedAt = time.Now()
}

func (d *DAG) IsDone() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, t := range d.tasks {
		switch t.Status {
		case StatusPending, StatusReady, StatusRunning:
			return false
		}
	}
	return true
}

func (d *DAG) Outputs() map[string]any {
	d.mu.RLock()
	defer d.mu.RUnlock()
	outputs := make(map[string]any)
	for id, t := range d.tasks {
		if t.Output != nil {
			outputs[id] = t.Output
		}
	}
	return outputs
}

func (d *DAG) FailedTasks() []*Task {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var failed []*Task
	for _, t := range d.tasks {
		if t.Status == StatusFailed {
			failed = append(failed, t)
		}
	}
	return failed
}

func (d *DAG) GetTask(id string) (*Task, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	t, ok := d.tasks[id]
	return t, ok
}

func (d *DAG) TopologicalSort() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	inDegree := make(map[string]int)
	for id := range d.tasks {
		inDegree[id] = 0
	}
	for _, t := range d.tasks {
		inDegree[t.ID] = len(t.DependsOn)
	}

	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	sort.Strings(queue)

	var result []string
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		result = append(result, id)

		for _, t := range d.tasks {
			for _, dep := range t.DependsOn {
				if dep == id {
					inDegree[t.ID]--
					if inDegree[t.ID] == 0 {
						queue = append(queue, t.ID)
						sort.Strings(queue)
					}
				}
			}
		}
	}

	return result
}

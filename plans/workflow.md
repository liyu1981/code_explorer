# `workflow` Package Plan — Plan-Execute-Evaluator Agent Loop

A self-contained package inside your existing Go app. Plugs into your existing LLM interface via a thin adapter.

> **Note**: This project already has a full `Agent` implementation in `pkg/agent` with tool execution, retry logic, and context management. The workflow package should integrate with (not duplicate) the existing `Agent` and `LLM` interfaces.

---

## Package Layout

```
pkg/workflow/
├── pee_workflow.go            # Plan-Execute-Evaluate workflow runner
├── dag.go                     # Task, DAG, status types
├── planner.go                 # Planner interface + LLM implementation
├── executor.go                # Parallel executor
├── evaluator.go               # Evaluator interface + LLM implementation
├── react_workflow.go          # ReAct workflow runner (Draft → Act → Observe)
├── rc_workflow.go             # Reflect-Critic workflow runner (Draft → Critique → Revise)
├── dynamic_workflow.go        # Dynamic routing (Switchboard) - routes to appropriate workflow
├── *_test.go                 # Unit tests
└── *_integration_test.go      # Integration tests
```

---

## Integration with Existing Code

The workflow package directly uses existing interfaces from `pkg/agent`:

| Original Plan | This Project |
|--------------|--------------|
| Custom `LLM.Complete(system, user)` | `agent.LLM.Generate(messages, tools, format)` |
| `ToolFunc func(ctx, input) (any, error)` | `agent.Tool` interface |
| Simple string prompts | `agent.Message` list |

---

## 1. `dag.go` — Core Data Structures

(Same as original, no changes needed)

```go
package workflow

import (
    "sync"
    "time"
)

type TaskStatus string

const (
    StatusPending  TaskStatus = "pending"
    StatusReady    TaskStatus = "ready"
    StatusRunning  TaskStatus = "running"
    StatusDone     TaskStatus = "done"
    StatusFailed   TaskStatus = "failed"
    StatusSkipped  TaskStatus = "skipped"
)

type Task struct {
    ID          string
    Description string
    Tool        string         // tool name to invoke (must exist in registry)
    Input       map[string]any // tool arguments; may reference other task IDs
    DependsOn   []string       // IDs of tasks that must be Done first

    Status      TaskStatus
    Output      any
    Err         error
    StartedAt   time.Time
    FinishedAt  time.Time
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

// validate checks all DependsOn refs exist and there are no cycles.
func (d *DAG) validate() error { /* topological sort */ }

// ReadyTasks returns Pending tasks whose every dependency is Done.
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

// IsDone returns true when every task is Done, Failed, or Skipped.
func (d *DAG) IsDone() bool { /* range tasks, check statuses */ }

// Outputs returns a snapshot of all task outputs keyed by task ID.
func (d *DAG) Outputs() map[string]any { /* range tasks, collect Output */ }

// FailedTasks returns tasks that ended in Failed.
func (d *DAG) FailedTasks() []*Task { /* filter */ }
```

---

## 3. `planner.go` — Planner

```go
package workflow

import (
    "context"
    "encoding/json"
    "fmt"

    "yourapp/pkg/agent"
)

// Planner turns a goal (plus optional failure context) into a DAG.
type Planner interface {
    Plan(ctx context.Context, req PlanRequest) (*DAG, error)
}

type PlanRequest struct {
    Goal           string
    Iteration      int
    PriorOutputs   map[string]any // reuse across replans
    FailedTasks    []*Task        // nil on first plan
    MissingInfo    []string       // hints from evaluator
}

// LLMPlanner is the default implementation backed by pkg/agent.LLM.
type LLMPlanner struct {
    llm            agent.LLM
    tools          []map[string]any
    responseFormat *agent.ResponseFormat
}

func NewLLMPlanner(llm agent.LLM, tools []map[string]any, responseFormat *agent.ResponseFormat) *LLMPlanner {
    return &LLMPlanner{
        llm:            llm,
        tools:          tools,
        responseFormat: responseFormat,
    }
}

func (p *LLMPlanner) Plan(ctx context.Context, req PlanRequest) (*DAG, error) {
    system := p.buildSystemPrompt()
    user   := p.buildUserPrompt(req)

    messages := []agent.Message{
        {Role: "system", Content: system},
        {Role: "user", Content: user},
    }

    raw, _, err := p.llm.Generate(ctx, messages, p.tools, p.responseFormat)
    if err != nil {
        return nil, fmt.Errorf("planner llm: %w", err)
    }

    var parsed struct {
        Tasks []struct {
            ID          string         `json:"id"`
            Description string         `json:"description"`
            Tool        string         `json:"tool"`
            Input       map[string]any `json:"input"`
            DependsOn   []string       `json:"depends_on"`
        } `json:"tasks"`
    }
    if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
        return nil, fmt.Errorf("planner parse: %w", err)
    }

    tasks := make([]*Task, len(parsed.Tasks))
    for i, pt := range parsed.Tasks {
        tasks[i] = &Task{
            ID:          pt.ID,
            Description: pt.Description,
            Tool:        pt.Tool,
            Input:       pt.Input,
            DependsOn:   pt.DependsOn,
            Status:      StatusPending,
        }
    }
    return NewDAG(tasks)
}

func (p *LLMPlanner) buildSystemPrompt() string {
    return `You are a task planner. Given a goal, output ONLY a JSON task graph.
Tasks may depend on each other. Return:
{"tasks":[{"id":"...","description":"...","tool":"...","input":{...},"depends_on":["..."]}]}`
}

func (p *LLMPlanner) buildUserPrompt(req PlanRequest) string {
    // include req.Goal, req.FailedTasks, req.MissingInfo, req.PriorOutputs summary
    return fmt.Sprintf("Goal: %s\nIteration: %d", req.Goal, req.Iteration)
}
```

---

## 4. `executor.go` — Parallel Executor

Uses existing `agent.ToolRegistry` and `agent.Tool` interface.

```go
package workflow

import (
    "context"
    "encoding/json"
    "sync"
    "time"

    "yourapp/pkg/agent"
)

type Executor struct {
    toolRegistry *agent.ToolRegistry
    maxWorkers  int
}

func NewExecutor(toolRegistry *agent.ToolRegistry, maxWorkers int) *Executor {
    return &Executor{toolRegistry: toolRegistry, maxWorkers: maxWorkers}
}

func (e *Executor) Execute(ctx context.Context, d *DAG) error {
    sem     := make(chan struct{}, e.maxWorkers)
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
                defer func() { <-sem; results <- task.ID }()

                tool, ok := e.toolRegistry.Get(task.Tool)
                if !ok {
                    d.SetResult(task.ID, nil, fmt.Errorf("unknown tool: %s", task.Tool))
                    d.SetStatus(task.ID, StatusFailed)
                    e.skipDependents(d, task.ID)
                    return
                }

                inputJSON, _ := json.Marshal(task.Input)
                output, err := tool.Execute(ctx, inputJSON, nil)
                d.SetResult(task.ID, output, err)
                if err != nil {
                    d.SetStatus(task.ID, StatusFailed)
                    e.skipDependents(d, task.ID)
                } else {
                    d.SetStatus(task.ID, StatusDone)
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

// skipDependents marks all tasks that (transitively) depend on failedID as Skipped.
func (e *Executor) skipDependents(d *DAG, failedID string) {
    // BFS/DFS over d.tasks: if any DependsOn contains failedID (or a skipped task), mark Skipped
}
```

---

## 5. `evaluator.go` — Evaluator

```go
package workflow

import (
    "context"
    "encoding/json"
    "fmt"

    "yourapp/pkg/agent"
)

type EvalStatus string

const (
    EvalDone   EvalStatus = "done"
    EvalReplan EvalStatus = "replan"
    EvalFailed EvalStatus = "failed"
)

type EvalResult struct {
    Status      EvalStatus
    FinalAnswer string
    ReplanHint  string
    MissingInfo []string
}

type Evaluator interface {
    Evaluate(ctx context.Context, goal string, d *DAG) (*EvalResult, error)
}

type LLMEvaluator struct {
    llm            agent.LLM
    tools          []map[string]any
    responseFormat *agent.ResponseFormat
    lastHint       string
}

func NewLLMEvaluator(llm agent.LLM, tools []map[string]any, responseFormat *agent.ResponseFormat) *LLMEvaluator {
    return &LLMEvaluator{
        llm:            llm,
        tools:          tools,
        responseFormat: responseFormat,
    }
}

func (ev *LLMEvaluator) Evaluate(ctx context.Context, goal string, d *DAG) (*EvalResult, error) {
    system := `You are a result evaluator. Given a goal and task outcomes, decide:
- "done": goal fully achieved, provide final_answer
- "replan": goal not met, explain replan_hint and missing_info
- "failed": unrecoverable error
Return ONLY JSON: {"status":"...","final_answer":"...","replan_hint":"...","missing_info":["..."]}`

    user := ev.buildPrompt(goal, d)

    messages := []agent.Message{
        {Role: "system", Content: system},
        {Role: "user", Content: user},
    }

    raw, _, err := ev.llm.Generate(ctx, messages, ev.tools, ev.responseFormat)
    if err != nil {
        return nil, fmt.Errorf("evaluator llm: %w", err)
    }

    var res EvalResult
    if err := json.Unmarshal([]byte(raw), &res); err != nil {
        return nil, fmt.Errorf("evaluator parse: %w", err)
    }

    if res.Status == EvalReplan && res.ReplanHint == ev.lastHint {
        res.Status = EvalFailed
        res.ReplanHint = "stagnation detected: " + res.ReplanHint
    }
    ev.lastHint = res.ReplanHint

    return &res, nil
}

func (ev *LLMEvaluator) buildPrompt(goal string, d *DAG) string {
    return fmt.Sprintf("Goal: %s\nTasks: <summarised>", goal)
}
```

---

## 6. `workflow.go` — Public Entry Point

```go
package workflow

import (
    "context"
    "fmt"

    "yourapp/pkg/agent"
)

type Runner struct {
    planner      Planner
    executor     *Executor
    evaluator    Evaluator
    maxIter      int
    toolRegistry *agent.ToolRegistry
}

func NewRunner(llm agent.LLM, toolRegistry *agent.ToolRegistry, maxWorkers, maxIter int) *Runner {
    tools := toolRegistry.MarshalToolsForLLM()
    return &Runner{
        planner:      NewLLMPlanner(llm, tools, nil),
        executor:     NewExecutor(toolRegistry, maxWorkers),
        evaluator:    NewLLMEvaluator(llm, tools, nil),
        maxIter:      maxIter,
        toolRegistry: toolRegistry,
    }
}

// Run is the only method callers need.
func (r *Runner) Run(ctx context.Context, goal string) (string, error) {
    req := PlanRequest{Goal: goal}

    for i := range r.maxIter {
        req.Iteration = i + 1

        dag, err := r.planner.Plan(ctx, req)
        if err != nil {
            return "", fmt.Errorf("iter %d plan: %w", req.Iteration, err)
        }

        if err := r.executor.Execute(ctx, dag); err != nil {
            return "", fmt.Errorf("iter %d execute: %w", req.Iteration, err)
        }

        result, err := r.evaluator.Evaluate(ctx, goal, dag)
        if err != nil {
            return "", fmt.Errorf("iter %d evaluate: %w", req.Iteration, err)
        }

        switch result.Status {
        case EvalDone:
            return result.FinalAnswer, nil
        case EvalFailed:
            return "", fmt.Errorf("unrecoverable: %s", result.ReplanHint)
        case EvalReplan:
            req.PriorOutputs = dag.Outputs()
            req.FailedTasks  = dag.FailedTasks()
            req.MissingInfo  = result.MissingInfo
        }
    }

    return "", fmt.Errorf("exceeded max iterations (%d)", r.maxIter)
}
```

---

## Alternative: Reuse Existing Agent

Instead of the executor using `agent.Tool` directly, consider wrapping each task in `agent.Agent`:

```go
// Option: Use agent.Agent for each task execution
// This provides retry, context management, and streaming support per task

type AgentTaskExecutor struct {
    agentConfig agent.AgentConfig
}

func (e *AgentTaskExecutor) Execute(ctx context.Context, input map[string]any) (string, error) {
    agent, err := agent.NewAgentFromConfig(ctx, &e.agentConfig)
    if err != nil {
        return "", err
    }
    return agent.Run(ctx, input["goal"].(string), nil, nil)
}
```

This approach:
- Reuses full `pkg/agent` capabilities (retry, streaming, context management)
- Executor becomes a simple dispatcher to agent per task
- Higher overhead per task but more robust execution

---

## Caller Usage

### PEE Workflow

```go
import (
    "yourapp/pkg/agent"
    "yourapp/pkg/workflow"
)

toolRegistry := agent.GetGlobalToolRegistry()
llm, _ := agent.BuildLLM(map[string]any{
    "type": "openai",
    "model": "qwen3.5:4b",
    "base_url": "http://localhost:11434/v1",
})

runner, _ := workflow.NewRunnerWithJSONFormat(llm, toolRegistry, 5, 10)
answer, err := runner.Run(ctx, "Summarize the latest Go release notes")
```

### ReAct Workflow

```go
import (
    "yourapp/pkg/agent"
    "yourapp/pkg/workflow"
)

toolRegistry := agent.GetGlobalToolRegistry()
llm, _ := agent.BuildLLM(map[string]any{
    "type": "openai",
    "model": "qwen3.5:4b",
    "base_url": "http://localhost:11434/v1",
})

runner := workflow.NewReactRunner(llm, toolRegistry,
    workflow.ReactWithMaxIterations(10),
    workflow.ReactWithMaxRetry(3),
)
answer, err := runner.Run(ctx, "Use the echo tool to say hello")
```

---

## What's Left to Implement

| Location | What | Status |
|---|---|---|
| `dag.go` | All methods implemented | Done |
| `executor.go` | All methods implemented | Done |
| `planner.go` | All methods implemented | Done |
| `evaluator.go` | All methods implemented | Done |
| `pee_workflow.go` | All methods implemented | Done |
| `react_workflow.go` | All methods implemented | Done |
| `rc_workflow.go` | All methods implemented | Done |
| `dynamic_workflow.go` | Intent detection + routing to appropriate workflow | Done |

---

## Relationship to Existing `pkg/agent`

The workflow package is **orthogonal** to the existing agent:

- `pkg/agent`: Single-turn or multi-turn conversation with tool execution
- `internal/workflow`: Meta-level planner that generates DAGs of tasks, then executes them

They can be composed:
1. Use workflow for complex multi-step tasks that need planning
2. Each workflow task can use an `agent.Agent` for its execution
3. Or, use workflow's executor directly with the existing tool registry

---

## ReAct Workflow

A standalone ReAct (Reasoning + Acting) workflow that interleaves model thinking with tool execution. Implemented in `pkg/workflow/react_workflow.go`.

### When to Use Each

| Pattern | Use Case |
|---------|----------|
| **ReAct** | Single complex goal, exploratory tasks, when you want the model to figure out steps as it goes |
| **PEE** | Complex multi-step tasks with dependencies, when upfront planning is beneficial, reusable task graphs |

### Implementation (`pkg/workflow/react_workflow.go`)

```go
type ReactRunner struct {
    llm            agent.LLM
    toolRegistry   *agent.ToolRegistry
    systemPrompt   string
    maxIterations  int
    maxRetry       int
    responseFormat *agent.ResponseFormat
    messages       []agent.Message
}

// Options
func ReactWithMaxIterations(n int) ReactRunnerOption
func ReactWithMaxRetry(n int) ReactRunnerOption
func ReactWithSystemPrompt(prompt string) ReactRunnerOption
func ReactWithResponseFormat(rf *agent.ResponseFormat) ReactRunnerOption

// Create runner
func NewReactRunner(llm agent.LLM, toolRegistry *agent.ToolRegistry, opts ...ReactRunnerOption) *ReactRunner

// Run workflow
func (r *ReactRunner) Run(ctx context.Context, goal string) (string, error)
func (r *ReactRunner) RunWithStream(ctx context.Context, goal string, stream protocol.IStreamWriter) (string, error)
```

### ReAct Loop

```
┌─────────────────────────────────────┐
│  1. System + User Message           │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  2. LLM Generate                    │
│     (with tools)                    │
└──────────────┬──────────────────────┘
               │
        ┌──────┴──────┐
        │              │
        ▼              ▼
   ┌─────────┐   ┌─────────────────┐
   │ No Tools │   │ Has Tool Calls │
   │ Return   │   └────────┬────────┘
   └─────────┘            │
                          ▼
               ┌─────────────────────┐
               │  Execute Tools      │
               │  (in parallel)     │
               └────────┬───────────┘
                        │
                        ▼
               ┌─────────────────────┐
               │  Add Tool Results  │
               │  as Messages       │
               └────────┬───────────┘
                        │
                        └──► Back to LLM Generate
```

### ReAct vs PEE Comparison

| Aspect | ReAct | PEE |
|--------|-------|-----|
| Planning | Dynamic (during execution) | Upfront (before execution) |
| Loop | Think → Act → Observe | Plan → Execute → Evaluate |
| Dependencies | Implicit in tool usage | Explicit in DAG |
| Best for | Exploratory, unknown steps | Structured, known steps |
| Complexity | Lower | Higher |
| Debugging | Easier to trace | Need to inspect DAG |

### ReAct as PEE Task

ReAct can be used as a single task within a PEE workflow:

```go
// Create a ReAct runner for a specific task
reactRunner := workflow.NewReactRunner(llm, toolRegistry,
    workflow.ReactWithMaxIterations(10),
)

// The executor can call it like any other tool
// (wrap in a Tool implementation if needed)
```

---

## Reflect-Critic Workflow

A Generator-Critic loop where the model drafts a response, then critiques itself, then revises. Particularly effective with Qwen models which tend to be "chatty" and reflective.

### Why Reflect-Critic?

- **Qwen models are naturally reflective**: They produce verbose "thinking" that can be captured as critiques
- **Self-correction**: The model catches its own errors before they propagate
- **Better tool use**: Catches missing parameters, hallucinated paths, wrong tool selection
- **Improved accuracy**: Multiple passes refine the output quality

### Workflow Loop

```
┌─────────────────────────────────────┐
│  1. Generate Draft                  │
│     (tool calls or text answer)     │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  2. Execute Tool(s)                 │
│     (if tool calls were made)       │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  3. Self-Critique                  │
│     "Critique your tool call for    │
│      potential errors, missing      │
│      parameters, or issues"         │
└──────────────┬──────────────────────┘
               │
        ┌──────┴──────┐
        │              │
        ▼              ▼
   ┌─────────┐   ┌────────────┐
   │ Issues  │   │ No Issues  │
   │ Found   │   │ Continue   │
   └────┬────┘   └────────────┘
        │
        ▼
┌─────────────────────────────────────┐
│  4. Revise / Retry                  │
│     (fix issues, re-execute)       │
└──────────────┬──────────────────────┘
               │
               └──► Back to Step 2 (if needed)
```

### When to Use

| Scenario | Recommended Pattern |
|----------|-------------------|
| Complex tool calls with many parameters | Reflect-Critic |
| Qwen models (chatty/reflective) | Reflect-Critic |
| High-stakes outputs requiring verification | Reflect-Critic |
| Simple, single-step tasks | ReAct |
| Exploratory tasks with unknown steps | ReAct |

### Implementation Concept

```go
type ReflectCriticRunner struct {
    llm            agent.LLM
    toolRegistry   *agent.ToolRegistry
    maxReflections int      // Max critique-revise cycles
    maxIterations  int      // Max tool call iterations
}

func NewReflectCriticRunner(llm agent.LLM, toolRegistry *agent.ToolRegistry) *ReflectCriticRunner

func (r *ReflectCriticRunner) Run(ctx context.Context, goal string) (string, error) {
    // Loop: Generate → Execute → Critique → Revise
}
```

### Critique Prompts

**System Prompt (Generator)**:
```
You are a precise assistant. Make tool calls only when necessary.
```

**Critique Prompt**:
```
Review your previous tool call and tool results:
1. Were all required parameters provided?
2. Were the parameter values correct?
3. Were there any errors in the results?
4. Is the task complete, or do you need more information?

Respond with your critique.
```

### Reflect-Critic vs Other Patterns

| Aspect | ReAct | PEE | Reflect-Critic |
|--------|-------|-----|----------------|
| Self-correction | No | Optional (via evaluator) | Yes (built-in) |
| Loop structure | Act → Observe | Plan → Execute → Evaluate | Draft → Critique → Revise |
| Best for | Simple tasks | Complex tasks with dependencies | High-accuracy tool use |
| Model fit | All models | All models | Qwen / chatty models |

---

## Dynamic Routing Workflow (Switchboard)

A meta-level router that analyzes user intent and dispatches to the appropriate workflow pattern.

### Intent Detection Flow

```
┌─────────────────────────────────────┐
│  User Request                       │
└──────────────┬──────────────────────┘
               │
               ▼
┌─────────────────────────────────────┐
│  Intent Detection                   │
│  (simple LLM call, no tools)        │
└──────────────┬──────────────────────┘
               │
        ┌──────┼──────┬──────────────┐
        │             │              │
        ▼             ▼              ▼
   ┌─────────┐  ┌─────────┐  ┌─────────────┐
   │ Simple  │  │Invest-  │  │  Complex    │
   │Question │  │igation  │  │  Report     │
   └────┬────┘  └────┬────┘  └──────┬──────┘
        │             │               │
        ▼             ▼               ▼
   ┌─────────┐  ┌─────────┐  ┌─────────────┐
   │  Direct │  │  ReAct  │  │    PEE      │
   │  LLM    │  │  Loop   │  │   (Plan)    │
   └─────────┘  └─────────┘  └─────────────┘
```

### Routing Logic

| Intent | Indicators | Workflow |
|--------|-----------|---------|
| **Simple Question** | "what is", "who is", "define", "explain" | Direct LLM response |
| **Code Investigation** | "find", "search", "read", "grep", "analyze" | ReAct or Reflect-Critic |
| **Complex Report** | "compare", "summarize", "generate", "report", multi-step | PEE (Plan-Execute-Evaluate) |

### Implementation Concept

```go
type Intent string

const (
    IntentSimple      Intent = "simple"
    IntentInvestgate  Intent = "investigate"
    IntentComplex    Intent = "complex"
)

type RouteResult struct {
    Intent       Intent
    Confidence   float64
    SuggestedWorkflow string
    Reasoning    string
}

type DynamicRouter struct {
    llm           agent.LLM
    toolRegistry  *agent.ToolRegistry
    simpleRunner  *SimpleRunner      // Direct LLM response
    reactRunner   *ReactRunner       // ReAct loop
    peeRunner     *PEEWorkflowRunner // Plan-Execute-Evaluate
    rcRunner      *RCRunner          // Reflect-Critic
}

func NewDynamicRouter(llm agent.LLM, toolRegistry *agent.ToolRegistry) *DynamicRouter

func (r *DynamicRouter) Route(ctx context.Context, goal string) (*RouteResult, error)

func (r *DynamicRouter) Run(ctx context.Context, goal string) (string, error) {
    // 1. Detect intent
    route, err := r.Route(ctx, goal)
    if err != nil {
        return "", err
    }

    // 2. Dispatch to appropriate workflow
    switch route.Intent {
    case IntentSimple:
        return r.simpleRunner.Run(ctx, goal)
    case IntentInvestigate:
        return r.reactRunner.Run(ctx, goal)
    case IntentComplex:
        return r.peeRunner.Run(ctx, goal)
    }
}
```

### Intent Detection Prompt

```json
{
  "type": "json_schema",
  "json_schema": {
    "name": "intent_detection",
    "schema": {
      "type": "object",
      "properties": {
        "intent": {
          "type": "string",
          "enum": ["simple", "investigate", "complex"],
          "description": "simple: factual questions, investigate: code/file search, complex: multi-step reports"
        },
        "confidence": {"type": "number"},
        "reasoning": {"type": "string"}
      },
      "required": ["intent", "confidence", "reasoning"]
    }
  }
}
```

### Fallback Strategy

If intent detection is uncertain (confidence < threshold), default to the more capable workflow (PEE) to ensure quality results.

```go
const MinConfidenceThreshold = 0.7

if route.Confidence < MinConfidenceThreshold {
    // Default to PEE for uncertain cases
    return r.peeRunner.Run(ctx, goal)
}
```

### When to Use Dynamic Routing

| Scenario | Dynamic Routing | Single Pattern |
|----------|----------------|----------------|
| Mixed workload (Q&A + code investigation + reports) | Yes | No |
| Single use case | No | Yes |
| Need optimal resource usage | Yes | No |
| Simpler implementation preferred | No | Yes |

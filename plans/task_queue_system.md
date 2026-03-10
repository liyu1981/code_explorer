# Background Task Queue System

## Goal
Implement a persistent, SQLite-backed task queue to handle long-running background jobs like codebase indexing, wiki generation, and others. The system will support progress tracking and real-time notifications to the frontend.

## Architecture

### 1. Database Schema
A new `tasks` table will be added via migration:
```sql
CREATE TABLE IF NOT EXISTS tasks (
    id          TEXT PRIMARY KEY,       -- NanoID
    name        TEXT NOT NULL,          -- Task type (e.g., 'codemogger-index')
    payload     TEXT NOT NULL,          -- JSON configuration for the task
    status      TEXT NOT NULL DEFAULT 'pending', -- pending, running, completed, failed
    progress    INTEGER NOT NULL DEFAULT 0,      -- 0-100
    message     TEXT,                   -- Latest status message or error
    retries     INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME
);
```

### 2. Backend Implementation (`pkg/task`)
- **`Manager`**: Responsible for submitting tasks, claiming them (atomic transaction), and updating status/progress.
- **`Worker`**: A pool of goroutines that poll the database for pending tasks.
- **`Registry`**: A mapping of task names to handler functions.
- **`TaskHandler`**: Interface or function type: `func(ctx context.Context, task *Task) error`.

### 3. Progress Tracking & Notifications
- Handlers will update task progress/message via the `Manager`.
- **Measurement Strategy**:
    - **Deterministic (e.g., File Indexing)**: Progress is calculated as `(processed_items / total_items) * 100`. The handler first performs a "count" or "scan" phase to establish the denominator.
    - **Step-based (e.g., Wiki Generation)**: Progress is mapped to discrete phases (e.g., "Scanning" = 10%, "Analyzing" = 40%, "Generating Markdown" = 80%, "Saving" = 100%).
    - **Indeterminate**: For tasks where the total volume is unknown, the system will use a "heartbeat" status message (e.g., "Processed 500 nodes...") with progress stuck at a symbolic value (e.g., 0 or -1) until completion.
- **Throttling**: To avoid overwhelming the database and WebSocket hub, the `Manager` will only persist/broadcast progress updates if the percentage has changed or if a minimum time interval (e.g., 500ms) has elapsed since the last update.
- **WebSocket Integration**:
    - The `Manager` will publish to the `WsHub` under the topic `tasks`.
    - Payload: `{ taskId: string, name: string, status: string, progress: number, message: string, timestamp: number }`.
- Frontend will subscribe to the `tasks` topic to receive real-time updates.

### 4. Management UI
- **Navigation Refactor**:
    - Introduce a "More" button in the primary sidebar.
    - Upon clicking "More", reveal a secondary menu (dropdown or slide-out) containing:
        - **Saved Reports**: Quick access to bookmarked snapshots.
        - **Sessions**: Historical research session management.
        - **Tasks**: The new background task monitor.
    - This keeps the primary navigation focused on "New" and "Active Research" while providing easy access to management tools.
- **Task Management Page**: `/tasks` to view the task list (paginated).
- **Details**: View task payload, error logs, and progress bars.
- **Actions**: Retry failed tasks, cancel pending/running tasks.

## Phase 1: Infrastructure
1.  **Migration**: Create `008_task_queue.up.sql`.
2.  **Queue Package**: Implement `pkg/task` with the core logic.
3.  **Integration**: Initialize the task queue in `pkg/server/server.go` and start workers.

## Phase 2: Refactoring Indexing
1.  **Task Handler**: Implement `codemogger-index` handler.
2.  **API Update**: Change `/api/codemogger/index` to submit a task instead of running it directly or using the current ad-hoc backgrounding.
3.  **WS Integration**: Ensure progress is broadcasted.

## Phase 3: Frontend Management
1.  **Task Store**: Create Jotai atoms for managing tasks.
2.  **Task List UI**: Build the management interface.
3.  **Global Notifications**: Add a "Task Center" or simple toast notifications for task completion/failure.

## Future Use Cases
- **Wiki Generation**: Long-running LLM jobs to document the codebase.
- **Batch Export**: Exporting research reports or data.
- **Codebase Sync**: Periodic syncing/re-scanning of codebases.

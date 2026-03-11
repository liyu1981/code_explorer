# Wiki Generation System Plan (v5: Persistent & Manageable Skill System)

This plan outlines a hierarchical agentic system to analyze diverse codebases using two agent behaviors and a persistent Skill system that users can customize.

## 1. Simplified Agent Architecture

### 1.1 Type A: The Worker (Iterative Executor)
- **Behavior**: Our current agent loop. It takes a specific goal, loads a **Skill** from the database, and iterates (Thinking -> Tool Use) until the goal is achieved.
- **Role**: Performs deep analysis of specific modules or directories.

### 1.2 Type B: The Orchestrator (Planner & Delegator)
- **Behavior**: An iterative agent with access to **Task System Tools**. It follows a "Plan -> Distribute -> Aggregate" loop.
- **Role**: Scans the root, creates sub-tasks for Workers, waits for completion, and assembles the final Knowledge Pages.

## 2. Persistent Skill System

The "Skill" system provides the system prompt and personality for an agent. To allow for user customization and persistence, skills are stored in the database but initialized from binary-embedded defaults.

### 2.1 Database Schema
A new `skills` table will be created:
- `id`: TEXT (Nanoid)
- `name`: TEXT (Unique, e.g., "go-expert")
- `description`: TEXT
- `system_prompt`: TEXT (The actual Markdown instructions)
- `is_builtin`: BOOLEAN (True if it came from the binary seeds)
- `created_at/updated_at`: DATETIME

### 2.2 Seeding & Lifecycle
1.  **Initial Seeding**: On application startup, the system reads embedded `.md` files from `pkg/prompt/skills/`.
2.  **Safe Sync (Skip if Exists)**: For each embedded skill, the system checks if a skill with the same name already exists in the database.
    -   If it **does not exist**: Insert the new skill into the DB.
    -   If it **already exists**: Skip the initialization for that skill. This ensures that user-revised versions of built-in skills are not overwritten by the binary defaults.
3.  **Loading**: When an agent task starts, the `task_handler` fetches the `system_prompt` from the database using the skill name specified in the task payload.

## 3. Tool & API Requirements

### 3.1 Codebase Discovery Tools
- **`get_tree`**: ASCII/Markdown directory structure representation.
- **`read_file`**: Read content with optional line range support.
- **`grep_search`**: Fast regex search across the project tree.

### 3.2 Task Delegation Tools (For Orchestrators)
- **`queue_task`**: Inserts a new task into the SQLite queue (specifying the target Skill name).
- **`poll_tasks`**: Checks the status of sub-tasks.
- **`read_task_output`**: Retrieves the final report produced by a completed sub-task.

### 3.3 Skill Management API
- `GET /api/skills`: List all available skills.
- `GET /api/skills/{name}`: Get detailed prompt for a skill.
- `PUT /api/skills/{name}`: Update a skill's system prompt (user override).
- `POST /api/skills/{name}/reset`: Revert a built-in skill to its embedded default.

## 4. Execution Flow: Wiki Generation

1.  **Entry**: User triggers "Generate Wiki". A `wiki-orchestrate` task is queued.
2.  **Orchestration (Type B)**:
    -   Loads `architect-planner` skill from DB.
    -   **Glance**: Uses `get_tree` to map the project.
    -   **Plan**: Identifies sub-modules and assigns skills (`go-expert`, `ts-frontend`).
    -   **Distribute**: Calls `queue_task` for each.
    -   **Wait**: Polls until sub-tasks are done.
3.  **Analysis (Type A)**:
    -   Workers pick up sub-tasks, load their assigned skills from the DB.
    -   Perform analysis and write a "Module Summary" to the task output.
4.  **Aggregation (Type B)**:
    -   Orchestrator resumes, aggregates all "Module Summaries" into the final Knowledge Base pages.

## 5. Milestones

1.  [ ] **DB Infrastructure**: Create `010_agent_skills.up.sql`, `010_agent_skills.down.sql` and `pkg/db/skill_store.go`.
2.  [ ] **Seeding**: Implement `go:embed` logic to sync default skills to the DB on startup.
3.  [ ] **Agent Refactor**: Update the Agent to fetch its system prompt via `skillStore.GetByName`.
4.  [ ] **Tools**: Implement discovery (`read_file`, `get_tree`) and delegation (`queue_task`) tools.
5.  [ ] **Management UI**: Create a "Skills" settings page to browse and edit prompts.
6.  [ ] **Orchestration**: Implement and test the full `wiki-orchestrate` -> `wiki-analyze` loop.

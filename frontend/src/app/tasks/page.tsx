"use client";

import {
  Activity,
  CheckCircle2,
  Clock,
  Loader2,
  AlertCircle,
} from "lucide-react";
import { useState } from "react";
import useSWR from "swr";
import { API_URL, fetcher } from "@/lib/api";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import { LoadingState } from "../_components/loading-state";
import { ErrorState } from "../_components/error-state";
import { EmptyState } from "../_components/empty-state";
import { Pagination } from "../_components/pagination";
import { TaskTable } from "./_components/task-table";
import { TaskDetailDialog } from "./_components/task-detail-dialog";

interface Task {
  id: string;
  name: string;
  payload: string;
  status: "pending" | "running" | "completed" | "failed";
  progress: number;
  message: { String: string; Valid: boolean };
  retries: number;
  max_retries: number;
  error?: { String: string; Valid: boolean };
  created_at: string;
  updated_at: string;
  completed_at?: { Time: string; Valid: boolean };
}

export default function TasksPage() {
  const [page, setPage] = useState(1);
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);
  const pageSize = 10;

  const { data, error, isLoading } = useSWR(
    `${API_URL}/api/tasks?page=${page}&pageSize=${pageSize}`,
    fetcher,
    { refreshInterval: 2000 },
  );

  const tasks = data?.tasks as Task[];
  const total = data?.total || 0;
  const totalPages = Math.ceil(total / pageSize);

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "completed":
        return <CheckCircle2 className="h-4 w-4 text-green-500" />;
      case "failed":
        return <AlertCircle className="h-4 w-4 text-destructive" />;
      case "running":
        return <Loader2 className="h-4 w-4 text-primary animate-spin" />;
      default:
        return <Clock className="h-4 w-4 text-muted-foreground" />;
    }
  };

  const getStatusClass = (status: string) => {
    switch (status) {
      case "completed":
        return "bg-green-500/10 text-green-500 border-green-500/20";
      case "failed":
        return "bg-destructive/10 text-destructive border-destructive/20";
      case "running":
        return "bg-primary/10 text-primary border-primary/20";
      default:
        return "bg-muted text-muted-foreground border-border";
    }
  };

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center gap-4">
          <Activity className="h-5 w-5 text-primary" />
          <h1 className="text-xl font-bold tracking-tight text-primary">
            Tasks
          </h1>
        </div>
      </AppHeader>

      <div className="flex-1 overflow-auto p-6 bg-background/50">
        <div className="max-w-6xl mx-auto w-full">
          <div className="flex items-center justify-between mb-6">
            <p className="text-muted-foreground text-sm">
              Monitor long-running background operations like codebase indexing.
            </p>
          </div>

          {isLoading && !tasks ? (
            <LoadingState />
          ) : error ? (
            <ErrorState title="Failed to load tasks" />
          ) : !tasks || tasks.length === 0 ? (
            <EmptyState
              icon={<Activity className="h-12 w-12" />}
              title="No tasks found."
              description="New tasks will appear here when started."
            />
          ) : (
            <>
              <TaskTable
                tasks={tasks}
                onTaskClick={setSelectedTask}
                getStatusIcon={getStatusIcon}
                getStatusClass={getStatusClass}
              />
              <Pagination
                page={page}
                totalPages={totalPages}
                totalItems={total}
                pageSize={pageSize}
                onPageChange={setPage}
                itemName="tasks"
              />
            </>
          )}
        </div>
      </div>

      <TaskDetailDialog
        task={selectedTask}
        onClose={() => setSelectedTask(null)}
        getStatusIcon={getStatusIcon}
        getStatusClass={getStatusClass}
      />
    </AppContainer>
  );
}

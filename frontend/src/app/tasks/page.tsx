"use client";

import { useAtom } from "jotai";
import {
  Activity,
  CheckCircle2,
  Clock,
  Loader2,
  AlertCircle,
  PlayCircle,
} from "lucide-react";
import { useState } from "react";
import useSWR from "swr";
import { API_URL, fetcher } from "@/lib/api";
import { cn } from "@/lib/utils";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";

interface Task {
  id: string;
  name: string;
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
  const pageSize = 10;

  const { data, error, isLoading, mutate } = useSWR(
    `${API_URL}/api/tasks?page=${page}&pageSize=${pageSize}`,
    fetcher,
    { refreshInterval: 2000 }, // Poll every 2 seconds
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
            <div className="flex items-center justify-center py-24">
              <Loader2 className="h-8 w-8 animate-spin text-primary/50" />
            </div>
          ) : error ? (
            <div className="text-center py-24 text-destructive bg-destructive/5 rounded-2xl border border-destructive/20">
              <AlertCircle className="h-8 w-8 mx-auto mb-4 opacity-50" />
              <p className="font-semibold">Failed to load tasks</p>
              <p className="text-sm opacity-70">
                Please check your connection and try again.
              </p>
            </div>
          ) : !tasks || tasks.length === 0 ? (
            <div className="text-center py-24 bg-muted/20 rounded-3xl border border-dashed border-border">
              <Activity className="h-12 w-12 mx-auto mb-4 text-muted-foreground/30" />
              <p className="text-muted-foreground font-medium">
                No tasks found.
              </p>
              <p className="text-sm text-muted-foreground/60">
                New tasks will appear here when started.
              </p>
            </div>
          ) : (
            <div className="bg-card border border-border rounded-2xl overflow-hidden shadow-sm">
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse">
                  <thead>
                    <tr className="bg-muted/30 border-b border-border">
                      <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
                        Task
                      </th>
                      <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
                        Status
                      </th>
                      <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
                        Progress
                      </th>
                      <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
                        Message
                      </th>
                      <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
                        Created
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border/50">
                    {tasks.map((task) => (
                      <tr
                        key={task.id}
                        className="hover:bg-muted/10 transition-colors"
                      >
                        <td className="px-6 py-4">
                          <div className="flex flex-col gap-0.5">
                            <span className="font-bold text-foreground capitalize">
                              {task.name.replace(/-/g, " ")}
                            </span>
                            <span className="text-[10px] font-mono text-muted-foreground">
                              {task.id}
                            </span>
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <div
                            className={cn(
                              "inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider border",
                              getStatusClass(task.status),
                            )}
                          >
                            {getStatusIcon(task.status)}
                            {task.status}
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <div className="w-32 flex flex-col gap-1.5">
                            <div className="flex items-center justify-between text-[10px] font-mono">
                              <span>{task.progress}%</span>
                            </div>
                            <div className="h-1.5 w-full bg-muted rounded-full overflow-hidden">
                              <div
                                className={cn(
                                  "h-full transition-all duration-500",
                                  task.status === "failed"
                                    ? "bg-destructive"
                                    : "bg-primary",
                                )}
                                style={{ width: `${task.progress}%` }}
                              />
                            </div>
                          </div>
                        </td>
                        <td className="px-6 py-4">
                          <div className="max-w-xs">
                            <p className="text-xs text-foreground line-clamp-2">
                              {(task.message?.Valid && task.message.String) ||
                                "No message"}
                            </p>
                            {task.status === "failed" && task.error?.Valid && (
                              <p className="text-[10px] text-destructive mt-1 font-mono line-clamp-1">
                                {task.error.String}
                              </p>
                            )}
                          </div>
                        </td>
                        <td className="px-6 py-4 text-xs text-muted-foreground font-mono">
                          {new Date(task.created_at).toLocaleString()}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              {totalPages > 1 && (
                <div className="px-6 py-4 border-t border-border bg-muted/20 flex items-center justify-between">
                  <span className="text-sm text-muted-foreground font-medium">
                    Page {page} of {totalPages} ({total} tasks)
                  </span>
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      disabled={page === 1}
                      onClick={() => setPage((p) => p - 1)}
                      className="px-4 py-1.5 rounded-lg border border-border bg-background text-sm font-bold disabled:opacity-50 hover:bg-muted transition-all active:scale-95"
                    >
                      Previous
                    </button>
                    <button
                      type="button"
                      disabled={page === totalPages}
                      onClick={() => setPage((p) => p + 1)}
                      className="px-4 py-1.5 rounded-lg border border-border bg-background text-sm font-bold disabled:opacity-50 hover:bg-muted transition-all active:scale-95"
                    >
                      Next
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </AppContainer>
  );
}

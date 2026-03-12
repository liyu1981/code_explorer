import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

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

interface TaskTableProps {
  tasks: Task[];
  onTaskClick: (task: Task) => void;
  getStatusIcon: (status: string) => ReactNode;
  getStatusClass: (status: string) => string;
}

export function TaskTable({
  tasks,
  onTaskClick,
  getStatusIcon,
  getStatusClass,
}: TaskTableProps) {
  return (
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
              <tr key={task.id} className="hover:bg-muted/10 transition-colors">
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
                <td
                  className="px-6 py-4 cursor-pointer hover:bg-primary/5 transition-colors group"
                  onClick={() => onTaskClick(task)}
                >
                  <div className="max-w-xs">
                    <p className="text-xs text-foreground line-clamp-2 group-hover:text-primary transition-colors">
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
    </div>
  );
}

import * as Dialog from "@radix-ui/react-dialog";
import { X, GitGraph } from "lucide-react";
import { Badge } from "../../_components/badge";
import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

interface Task {
  id: string;
  name: string;
  payload: string;
  initiator_id: { String: string; Valid: boolean };
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

interface TaskDetailDialogProps {
  task: Task | null;
  onClose: () => void;
  onViewLineage: (taskId: string) => void;
  getStatusIcon: (status: string) => ReactNode;
  getStatusClass: (status: string) => string;
}

export function TaskDetailDialog({
  task,
  onClose,
  onViewLineage,
  getStatusIcon,
  getStatusClass,
}: TaskDetailDialogProps) {
  const formatPayload = (payload: string) => {
    try {
      return JSON.stringify(JSON.parse(payload), null, 2);
    } catch {
      return payload;
    }
  };

  return (
    <Dialog.Root open={!!task} onOpenChange={(open) => !open && onClose()}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/40 backdrop-blur-sm z-[100]" />
        <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[90vw] max-w-2xl bg-card border border-border rounded-2xl shadow-2xl p-0 overflow-hidden z-[101] outline-none flex flex-col max-h-[90vh]">
          <div className="px-6 py-4 border-b border-border flex items-center justify-between bg-muted/30">
            <Dialog.Title className="text-sm font-bold text-foreground flex items-center gap-2">
              {task && getStatusIcon(task.status)}
              Task Details
            </Dialog.Title>
            <Dialog.Close className="p-1 rounded-lg hover:bg-muted transition-colors">
              <X className="h-4 w-4 text-muted-foreground" />
            </Dialog.Close>
          </div>
          <div className="p-6 overflow-auto">
            {task && (
              <div className="space-y-6">
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div className="space-y-1">
                    <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">
                      Task Name
                    </p>
                    <p className="font-bold capitalize text-primary">
                      {task.name.replace(/-/g, " ")}
                    </p>
                  </div>
                  <div className="space-y-1">
                    <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">
                      Status
                    </p>
                    <div
                      className={cn(
                        "inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider border",
                        getStatusClass(task.status),
                      )}
                    >
                      {task.status}
                    </div>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div className="space-y-1">
                    <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">
                      Initiator
                    </p>
                    {task.initiator_id?.Valid ? (
                      <div className="flex items-center gap-2">
                        <span className="font-mono text-[10px] text-primary font-bold">
                          {task.initiator_id.String}
                        </span>
                        <button
                          onClick={() =>
                            onViewLineage(task.initiator_id.String)
                          }
                          className="p-1 rounded-lg hover:bg-primary/10 text-primary transition-colors"
                          title="View Lineage of Initiator"
                        >
                          <GitGraph className="h-3 w-3" />
                        </button>
                      </div>
                    ) : (
                      <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest text-muted-foreground/60">
                        User
                      </span>
                    )}
                  </div>
                  <div className="space-y-1">
                    <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">
                      Progress
                    </p>
                    <span className="font-mono text-xs">{task.progress}%</span>
                  </div>
                </div>

                <div className="space-y-2">
                  <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">
                    Message
                  </p>
                  <div className="bg-muted/50 rounded-xl p-4 border border-border/50 text-sm whitespace-pre-wrap font-medium leading-relaxed">
                    {(task.message?.Valid && task.message.String) ||
                      "No message provided."}
                  </div>
                </div>

                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <p className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">
                      Payload
                    </p>
                    <button
                      onClick={() => onViewLineage(task.id)}
                      className="flex items-center gap-1 px-2 py-0.5 rounded-lg hover:bg-primary/10 text-primary transition-colors text-[10px] font-bold"
                    >
                      <GitGraph className="h-3 w-3" />
                      View Lineage
                    </button>
                  </div>
                  <div className="bg-muted/30 rounded-xl p-4 border border-border/50 text-xs font-mono overflow-auto max-h-[300px] whitespace-pre">
                    {formatPayload(task.payload)}
                  </div>
                </div>

                {task.status === "failed" && task.error?.Valid && (
                  <div className="space-y-2">
                    <p className="text-[10px] font-bold text-destructive uppercase tracking-wider">
                      Error Details
                    </p>
                    <div className="bg-destructive/5 rounded-xl p-4 border border-destructive/20 text-xs font-mono text-destructive break-all">
                      {task.error.String}
                    </div>
                  </div>
                )}

                <div className="pt-4 border-t border-border/50 flex items-center justify-between text-[10px] text-muted-foreground font-mono">
                  <span>
                    Created: {new Date(task.created_at).toLocaleString()}
                  </span>
                  <span>ID: {task.id}</span>
                </div>
              </div>
            )}
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

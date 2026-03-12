import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface EmptyStateProps {
  icon: ReactNode;
  title: string;
  description?: string;
  className?: string;
}

export function EmptyState({
  icon,
  title,
  description,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        "text-center py-24 bg-muted/20 rounded-3xl border border-dashed border-border",
        className,
      )}
    >
      <div className="flex justify-center mb-4 opacity-30">{icon}</div>
      <p className="text-muted-foreground font-medium">{title}</p>
      {description && (
        <p className="text-sm text-muted-foreground/60">{description}</p>
      )}
    </div>
  );
}

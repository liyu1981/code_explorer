import { AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";

interface ErrorStateProps {
  title?: string;
  message?: string;
  className?: string;
}

export function ErrorState({
  title = "Failed to load",
  message = "Please check your connection and try again.",
  className,
}: ErrorStateProps) {
  return (
    <div
      className={cn(
        "text-center py-24 text-destructive bg-destructive/5 rounded-2xl border border-destructive/20",
        className,
      )}
    >
      <AlertCircle className="h-8 w-8 mx-auto mb-4 opacity-50" />
      <p className="font-semibold">{title}</p>
      <p className="text-sm opacity-70">{message}</p>
    </div>
  );
}

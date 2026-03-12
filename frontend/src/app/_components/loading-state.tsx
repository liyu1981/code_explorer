import { Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

interface LoadingStateProps {
  className?: string;
}

export function LoadingState({ className }: LoadingStateProps) {
  return (
    <div
      className={cn("flex items-center justify-center py-24 w-full", className)}
    >
      <Loader2 className="h-8 w-8 animate-spin text-primary/50" />
    </div>
  );
}

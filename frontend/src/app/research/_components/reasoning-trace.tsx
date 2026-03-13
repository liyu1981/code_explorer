"use client";

import { Check, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";

export interface ReasoningStep {
  id: string;
  label: string;
  status: "pending" | "active" | "completed";
}

interface ReasoningTraceProps {
  steps: ReasoningStep[];
}

export function ReasoningTrace({ steps }: ReasoningTraceProps) {
  return (
    <div className="flex flex-col gap-3 mx-auto max-w-2xl">
      <h3 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider px-1">
        Researching Process
      </h3>
      <div className="space-y-2">
        {steps.map((step) => (
          <div
            key={step.id}
            className={cn(
              "flex items-center gap-3 py-1 transition-opacity",
              step.status === "pending" ? "opacity-30" : "opacity-100",
            )}
          >
            <div className="flex-shrink-0 w-5 h-5 flex items-center justify-center">
              {step.status === "active" ? (
                <Loader2 className="h-4 w-4 animate-spin text-primary" />
              ) : step.status === "completed" ? (
                <Check className="h-4 w-4 text-green-500" />
              ) : (
                <div className="h-2 w-2 rounded-full bg-muted-foreground/30" />
              )}
            </div>
            <span
              className={cn(
                "text-sm font-medium",
                step.status === "active"
                  ? "text-foreground"
                  : "text-muted-foreground",
              )}
            >
              {step.label}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

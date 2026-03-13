"use client";

import { cn } from "@/lib/utils";
import { ReasoningTrace } from "./reasoning-trace";

interface Step {
  id: string;
  label: string;
  status: "pending" | "active" | "completed";
}

interface FloatingThoughtProcessProps {
  steps: Step[];
  thoughtProcess?: string;
  isVisible: boolean;
}

export function FloatingThoughtProcess({
  steps,
  thoughtProcess,
  isVisible,
}: FloatingThoughtProcessProps) {
  if (!isVisible) return null;

  return (
    <div className="absolute top-[1rem] left-0 right-0 z-50 px-6 animate-in fade-in slide-in-from-top-4 duration-500 pointer-events-none">
      <div className="max-w-2xl mx-auto pointer-events-auto">
        <div className="bg-background/80 backdrop-blur-xl border border-primary/20 shadow-2xl rounded-2xl overflow-hidden shadow-primary/5">
          <div className="p-4 border-b border-border/50 bg-muted/30">
            <ReasoningTrace steps={steps} />
          </div>

          {thoughtProcess && (
            <div className="max-h-[200px] overflow-auto p-4 bg-muted/10">
              <h4 className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest mb-2 px-1">
                Granular Thought Process
              </h4>
              <pre className="text-xs font-mono whitespace-pre-wrap text-muted-foreground/70 leading-relaxed">
                {thoughtProcess}
              </pre>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

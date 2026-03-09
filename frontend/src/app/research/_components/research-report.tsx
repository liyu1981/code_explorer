"use client";

import {
  MessageSquare,
  Sparkles,
  Clock,
  Trash2,
  Copy,
  Check,
} from "lucide-react";
import { useEffect, useState } from "react";
import { Markdown } from "../../_components/markdown";
import { ResearchTurn } from "../../_jotai/research-store";
import { SourceCard } from "./source-card";

interface ResearchReportProps {
  turns: ResearchTurn[];
  onFollowUp?: (query: string) => void;
  onDeleteTurn?: (turnId: string) => void;
  isStreaming?: boolean;
}

export function ResearchReport({
  turns,
  onFollowUp,
  onDeleteTurn,
  isStreaming,
}: ResearchReportProps) {
  const [, setTick] = useState(0);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  // Update relative times every minute
  useEffect(() => {
    const timer = setInterval(() => setTick((t) => t + 1), 60000);
    return () => clearInterval(timer);
  }, []);

  const handleCopy = async (id: string, text: string) => {
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(text);
      } else {
        // Fallback for non-secure contexts
        const textArea = document.createElement("textarea");
        textArea.value = text;
        textArea.style.position = "fixed";
        textArea.style.left = "-9999px";
        textArea.style.top = "0";
        document.body.appendChild(textArea);
        textArea.focus();
        textArea.select();
        try {
          document.execCommand("copy");
        } catch (err) {
          console.error("Fallback copy failed", err);
        }
        document.body.removeChild(textArea);
      }
      setCopiedId(id);
      setTimeout(() => setCopiedId(null), 2000);
    } catch (err) {
      console.error("Failed to copy!", err);
    }
  };

  const getRelativeTime = (ts?: number) => {
    if (!ts) return "";
    const now = Date.now();
    const diff = now - ts;
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (seconds < 30) return "just now";
    if (seconds < 60) return `${seconds}s ago`;
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    return `${days}d ago`;
  };

  return (
    <div className="max-w-6xl mx-auto w-full py-8">
      {turns.map((turn, turnIndex) => (
        <div key={turn.id} className="w-full" data-turn-id={turn.id}>
          {turnIndex > 0 && (
            <div className="py-12">
              <div className="border-t border-border/60 w-full relative">
                <div className="absolute left-1/2 -translate-x-1/2 -translate-y-1/2 bg-background px-4 text-[10px] font-bold text-muted-foreground/40 uppercase tracking-[0.2em]">
                  Next Phase
                </div>
              </div>
            </div>
          )}

          <div className="flex flex-col lg:flex-row items-stretch gap-12 animate-in fade-in slide-in-from-bottom-4 duration-700 w-full">
            <div className="flex-1 space-y-8 min-w-0 flex flex-col">
              <div className="flex items-center gap-3 text-primary/70">
                <Sparkles className="h-5 w-5 flex-shrink-0" />
                <span className="text-lg font-bold tracking-tight text-foreground/80 flex-1">
                  {turn.query}
                </span>
                <div className="flex items-center gap-4">
                  <div className="flex flex-col items-end gap-1">
                    <span className="text-[10px] font-bold text-muted-foreground/30 uppercase tracking-widest whitespace-nowrap">
                      Turn #{turnIndex + 1}
                    </span>
                    <div className="flex items-center gap-2 text-[9px] font-bold text-muted-foreground/40 uppercase tracking-tighter whitespace-nowrap">
                      <Clock className="h-2.5 w-2.5" />
                      <span>
                        Updated:{" "}
                        {getRelativeTime(turn.updatedAt || turn.timestamp)} :
                        Created: {getRelativeTime(turn.timestamp)}
                      </span>
                    </div>
                  </div>
                  <div className="flex items-center gap-1">
                    <button
                      onClick={() => handleCopy(turn.id, turn.report)}
                      className="p-2 text-muted-foreground/30 hover:text-primary hover:bg-primary/10 rounded-lg transition-all"
                      title="Copy Markdown"
                    >
                      {copiedId === turn.id ? (
                        <Check className="h-4 w-4 text-green-500" />
                      ) : (
                        <Copy className="h-4 w-4" />
                      )}
                    </button>
                    {onDeleteTurn && !isStreaming && (
                      <button
                        onClick={() => onDeleteTurn(turn.id)}
                        className="p-2 text-muted-foreground/30 hover:text-destructive hover:bg-destructive/10 rounded-lg transition-all"
                        title="Delete Turn"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                </div>
              </div>

              <div className="flex-1">
                <Markdown
                  content={turn.report}
                  className="text-lg leading-relaxed text-foreground/90"
                />
              </div>

              {turnIndex === turns.length - 1 && !isStreaming && (
                <div className="pt-12 border-t border-border/40">
                  <div className="flex items-center gap-2 mb-6">
                    <MessageSquare className="h-4 w-4 text-muted-foreground/60" />
                    <h3 className="font-bold text-muted-foreground/60 uppercase tracking-[0.15em] text-[11px]">
                      Deepen Research
                    </h3>
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    {[
                      "Analyze performance implications",
                      "How is this tested?",
                      "Are there security concerns?",
                    ].map((q) => (
                      <button
                        key={q}
                        onClick={() => onFollowUp?.(q)}
                        className="text-left px-5 py-4 rounded-2xl border border-border/40 bg-muted/20 hover:bg-muted/40 hover:border-primary/30 transition-all text-sm font-semibold"
                      >
                        {q}
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>

            <aside className="w-full lg:w-80 flex-shrink-0 space-y-8">
              <div className="space-y-4">
                <h3 className="font-bold text-muted-foreground/60 uppercase tracking-[0.15em] text-[11px] px-1">
                  Source Material
                </h3>
                <div className="grid grid-cols-1 gap-4">
                  {turn.sources.map((s) => (
                    <SourceCard key={s.id} source={s} />
                  ))}
                </div>
              </div>
            </aside>
          </div>
        </div>
      ))}
    </div>
  );
}

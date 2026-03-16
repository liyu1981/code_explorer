"use client";

import { ChevronRight, FileCode } from "lucide-react";

export interface Source {
  id: string;
  path: string;
  score?: number;
  snippet?: string;
  start_line?: number;
  end_line?: number;
}

interface SourceCardProps {
  source: Source;
  onClick?: (source: Source) => void;
}

export function SourceCard({ source, onClick }: SourceCardProps) {
  return (
    <button
      type="button"
      onClick={() => onClick?.(source)}
      className="flex flex-col gap-2 p-3 rounded-lg border bg-card text-left hover:border-primary/50 transition-all hover:shadow-sm"
    >
      <div className="flex items-center gap-2 text-xs font-medium text-foreground/70 truncate">
        <FileCode className="h-3 w-3" />
        <span className="truncate">{source.path}</span>
      </div>
      {source.snippet && (
        <pre className="text-[10px] leading-tight font-mono text-muted-foreground/90 line-clamp-3 bg-muted/30 p-1.5 rounded-sm">
          {source.snippet}
        </pre>
      )}
    </button>
  );
}

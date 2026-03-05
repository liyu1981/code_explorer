"use client";

import { ChevronRight, FileCode } from "lucide-react";

export interface Source {
  id: string;
  path: string;
  score?: number;
  snippet?: string;
}

interface SourceCardProps {
  source: Source;
  onClick?: (source: Source) => void;
}

export function SourceCard({ source, onClick }: SourceCardProps) {
  return (
    <button
      onClick={() => onClick?.(source)}
      className="flex flex-col gap-2 p-3 rounded-lg border bg-card text-left hover:border-primary/50 transition-all hover:shadow-sm"
    >
      <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground truncate">
        <FileCode className="h-3 w-3" />
        <span className="truncate">{source.path}</span>
      </div>
      {source.snippet && (
        <pre className="text-[10px] leading-tight font-mono text-muted-foreground/80 line-clamp-3 bg-muted/30 p-1.5 rounded-sm">
          {source.snippet}
        </pre>
      )}
      <div className="flex items-center justify-between mt-auto pt-1">
        <span className="text-[10px] text-muted-foreground">
          Source #{source.id}
        </span>
        <ChevronRight className="h-3 w-3 text-muted-foreground/40" />
      </div>
    </button>
  );
}

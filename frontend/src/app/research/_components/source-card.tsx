"use client";

import { ChevronDown, ChevronRight, FileCode } from "lucide-react";
import { useState } from "react";

export interface Source {
  id: string;
  path: string;
  score?: number;
  snippet?: string;
  start_line?: number;
  end_line?: number;
}

export interface SourceGroup {
  path: string;
  snippets: Source[];
}

function getSnippetKey(source: Source): string {
  return `${source.start_line ?? ""}-${source.end_line ?? ""}`;
}

export function groupSourcesByPath(sources: Source[]): SourceGroup[] {
  const groups = new Map<string, Map<string, Source>>();
  for (const source of sources) {
    if (!groups.has(source.path)) {
      groups.set(source.path, new Map());
    }
    const snippetKey = getSnippetKey(source);
    const existing = groups.get(source.path)!;
    if (!existing.has(snippetKey)) {
      existing.set(snippetKey, source);
    }
  }
  return Array.from(groups.entries())
    .map(([path, snippetsMap]) => ({
      path,
      snippets: Array.from(snippetsMap.values()),
    }))
    .map((group) => ({
      ...group,
      snippets: group.snippets.sort(
        (a, b) => (a.start_line ?? 0) - (b.start_line ?? 0),
      ),
    }));
}

interface SourceCardProps {
  source: Source;
  onClick?: (source: Source) => void;
  showLineNumbers?: boolean;
}

export function SourceCard({
  source,
  onClick,
  showLineNumbers,
}: SourceCardProps) {
  const lineNumbers = source.start_line
    ? source.end_line
      ? `${source.start_line}-${source.end_line}`
      : `${source.start_line}`
    : null;

  return (
    <button
      type="button"
      onClick={() => onClick?.(source)}
      className="flex flex-col gap-2 p-3 rounded-lg border bg-card text-left hover:border-primary/50 transition-all hover:shadow-sm"
    >
      <div className="flex items-center gap-2 text-xs font-medium text-foreground/70 truncate">
        <FileCode className="h-3 w-3" />
        <span className="truncate">{source.path}</span>
        {lineNumbers && (
          <span className="text-muted-foreground/40 font-mono text-[10px]">
            ({lineNumbers})
          </span>
        )}
      </div>
      {source.snippet && (
        <pre className="text-[10px] leading-tight font-mono text-muted-foreground/90 line-clamp-3 bg-muted/30 p-1.5 rounded-sm">
          {showLineNumbers && source.start_line
            ? source.snippet
                .split("\n")
                .map((line, i) => {
                  const lineNum = source.start_line! + i;
                  return `${lineNum.toString().padStart(4)} | ${line}`;
                })
                .join("\n")
            : source.snippet}
        </pre>
      )}
    </button>
  );
}

interface SourceGroupCardProps {
  group: SourceGroup;
  onClick?: (source: Source) => void;
  defaultExpanded?: boolean;
  showLineNumbers?: boolean;
}

export function SourceGroupCard({
  group,
  onClick,
  defaultExpanded = false,
  showLineNumbers = true,
}: SourceGroupCardProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);

  return (
    <div className="rounded-lg border bg-card text-left overflow-hidden">
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-2 w-full p-3 hover:bg-muted/30 transition-colors"
      >
        {expanded ? (
          <ChevronDown className="h-3 w-3 text-muted-foreground/50" />
        ) : (
          <ChevronRight className="h-3 w-3 text-muted-foreground/50" />
        )}
        <FileCode className="h-3 w-3 text-muted-foreground/70" />
        <span className="flex-1 text-xs font-medium text-foreground/70 truncate text-left">
          {group.path}
        </span>
        <span className="text-[10px] text-muted-foreground/40">
          {group.snippets.length} snippet{group.snippets.length > 1 ? "s" : ""}
        </span>
      </button>
      {expanded && (
        <div className="border-t bg-muted/20">
          {group.snippets.map((source, index) => (
            <div
              key={source.id}
              className="p-3 border-b last:border-b-0 border-border/30"
            >
              <button
                type="button"
                onClick={() => onClick?.(source)}
                className="w-full text-left hover:opacity-80 transition-opacity"
              >
                {source.snippet && (
                  <pre className="text-[10px] leading-tight font-mono text-muted-foreground/90 bg-muted/30 p-2 rounded-sm overflow-x-auto">
                    {showLineNumbers && source.start_line
                      ? source.snippet
                          .split("\n")
                          .map((line, i) => {
                            const lineNum = source.start_line! + i;
                            return `${lineNum.toString().padStart(4)} | ${line}`;
                          })
                          .join("\n")
                      : source.snippet}
                  </pre>
                )}
                {source.score !== undefined && (
                  <div className="mt-1 text-[9px] text-muted-foreground/40">
                    relevance: {(source.score * 100).toFixed(0)}%
                  </div>
                )}
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

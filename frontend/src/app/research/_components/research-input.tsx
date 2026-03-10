"use client";

import { Search, Sparkles } from "lucide-react";
import type React from "react";
import { useState } from "react";
import { cn } from "@/lib/utils";

interface ResearchInputProps {
  onSearch: (query: string, deepSearch: boolean) => void;
  isCompact?: boolean;
}

export function ResearchInput({ onSearch, isCompact }: ResearchInputProps) {
  const [query, setQuery] = useState("");
  const [isDeepSearch, setIsDeepSearch] = useState(false);

  const handleSubmit = (e?: React.FormEvent) => {
    e?.preventDefault();
    if (query.trim()) {
      onSearch(query, isDeepSearch);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  return (
    <div
      className={cn(
        "w-full max-w-3xl mx-auto transition-all duration-500",
        isCompact ? "p-4" : "p-8",
      )}
    >
      <form
        onSubmit={handleSubmit}
        className={cn(
          "relative flex flex-col rounded-3xl border bg-card p-2 shadow-sm transition-all focus-within:ring-4 focus-within:ring-primary/10 focus-within:border-primary/40",
          !isCompact && "shadow-2xl p-4",
        )}
      >
        <textarea
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="What would you like to research in the codebase?"
          className="w-full min-h-[80px] max-h-[400px] resize-none bg-transparent p-4 text-lg outline-none placeholder:text-muted-foreground/60 leading-relaxed"
          rows={1}
        />

        <div className="flex items-center justify-between px-2 pb-2">
          <button
            type="button"
            onClick={() => setIsDeepSearch(!isDeepSearch)}
            className={cn(
              "flex items-center gap-2 rounded-xl px-4 py-2 text-sm font-bold transition-all hover:scale-105 active:scale-95",
              isDeepSearch
                ? "bg-primary/10 text-primary border border-primary/20 shadow-sm"
                : "bg-muted/50 text-muted-foreground hover:bg-muted border border-transparent",
            )}
          >
            <Sparkles
              className={cn("h-4 w-4", isDeepSearch && "animate-pulse")}
            />
            Deep Research
          </button>

          <button
            type="submit"
            disabled={!query.trim()}
            className="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary text-primary-foreground shadow-lg shadow-primary/20 hover:scale-110 active:scale-95 transition-all disabled:opacity-30 disabled:hover:scale-100"
          >
            <Search className="h-6 w-6" />
          </button>
        </div>
      </form>
    </div>
  );
}

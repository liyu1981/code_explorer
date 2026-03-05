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
          "relative flex flex-col gap-2 rounded-2xl border bg-card p-4 shadow-sm transition-all focus-within:ring-2 focus-within:ring-primary/20",
          !isCompact && "shadow-xl",
        )}
      >
        <textarea
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="What would you like to research in the codebase?"
          className="w-full min-h-[60px] max-h-[300px] resize-none bg-transparent p-2 text-lg outline-none placeholder:text-muted-foreground"
          rows={1}
        />

        <div className="flex items-center justify-between border-t pt-3">
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setIsDeepSearch(!isDeepSearch)}
              className={cn(
                "flex items-center gap-2 rounded-full px-3 py-1 text-xs font-medium transition-colors",
                isDeepSearch
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground hover:bg-muted/80",
              )}
            >
              <Sparkles className="h-3 w-3" />
              Deep Research
            </button>
          </div>

          <button
            type="submit"
            disabled={!query.trim()}
            className="flex h-10 w-10 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-sm hover:bg-primary/90 disabled:opacity-50"
          >
            <Search className="h-5 w-5" />
          </button>
        </div>
      </form>
    </div>
  );
}

"use client";

import { ResearchInput } from "./research-input";

interface StickyResearchInputProps {
  onSearch: (query: string, deep: boolean) => void;
  suggestions?: string[];
  isVisible: boolean;
}

export function StickyResearchInput({
  onSearch,
  suggestions = [],
  isVisible,
}: StickyResearchInputProps) {
  if (!isVisible) return null;

  return (
    <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-background via-background to-transparent pt-20 pb-8 px-6 z-20 pointer-events-none">
      <div className="max-w-6xl mx-auto pointer-events-auto">
        <ResearchInput
          onResearch={onSearch}
          isCompact
          suggestions={suggestions}
        />
      </div>
    </div>
  );
}

"use client";

import { ResearchInput } from "./research-input";

interface IdleResearchViewProps {
  onSearch: (query: string, deep: boolean) => void;
  title?: string;
  subtitle?: string;
}

export function IdleResearchView({
  onSearch,
  title = "What are we building?",
  subtitle = "Research your codebase with semantic intelligence and deep analytical reasoning.",
}: IdleResearchViewProps) {
  return (
    <div className="w-full max-w-4xl space-y-12 animate-in fade-in zoom-in-95 duration-700">
      <div className="text-center space-y-4">
        <h2 className="text-6xl font-bold tracking-tighter">{title}</h2>
        <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
          {subtitle}
        </p>
      </div>
      <div className="max-w-6xl mx-auto p-8">
        <ResearchInput onSearch={onSearch} />
      </div>
    </div>
  );
}

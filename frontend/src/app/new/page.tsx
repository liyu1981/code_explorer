"use client";

import { PlusCircle } from "lucide-react";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import { CodebaseList } from "./_components/codebase-list";

export default function NewResearchPage() {
  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center gap-4">
          <PlusCircle className="h-5 w-5 text-primary" />
          <h1 className="text-xl font-bold tracking-tight text-primary">
            New Research
          </h1>
        </div>
      </AppHeader>
      <div className="flex-1 overflow-auto p-6 md:p-12 bg-background/50">
        <div className="max-w-6xl mx-auto w-full">
          <CodebaseList />
        </div>
      </div>
    </AppContainer>
  );
}

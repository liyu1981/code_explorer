"use client";

import { useAtom } from "jotai";
import {
  ArrowRight,
  Clock,
  Database,
  Folder,
  Search as SearchIcon,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import {
  createSession,
  researchSessionsAtom,
} from "../../_jotai/research-store";

interface CodebaseMock {
  id: string;
  name: string;
  path: string;
  indexedAt: string;
}

const MOCK_CODEBASES: CodebaseMock[] = [
  {
    id: "1",
    name: "code_explorer",
    path: "/home/yli/code_explorer",
    indexedAt: "2026-03-05 10:00:00",
  },
  {
    id: "2",
    name: "nextjs-app",
    path: "/home/yli/projects/next-web",
    indexedAt: "2026-03-04 15:30:00",
  },
  {
    id: "3",
    name: "go-backend",
    path: "/home/yli/projects/go-api",
    indexedAt: "2026-03-03 09:15:00",
  },
  {
    id: "4",
    name: "react-components",
    path: "/home/yli/ui-lib",
    indexedAt: "2026-03-02 18:45:00",
  },
];

export function CodebaseList() {
  const [, setSessions] = useAtom(researchSessionsAtom);
  const [codebaseFilter, setCodebaseFilter] = useState("");
  const router = useRouter();

  const handleNewResearch = (codebase?: string) => {
    const newSession = createSession();
    if (codebase) {
      newSession.title = `Research: ${codebase}`;
    }
    setSessions((current) => [...current, newSession]);
    router.push(`/research?id=${newSession.id}`);
  };

  const filteredCodebases = MOCK_CODEBASES.filter(
    (c) =>
      c.name.toLowerCase().includes(codebaseFilter.toLowerCase()) ||
      c.path.toLowerCase().includes(codebaseFilter.toLowerCase()),
  );

  return (
    <div className="max-w-3xl mx-auto space-y-12">
      <div className="space-y-4 text-center">
        <h2 className="text-4xl font-bold tracking-tight text-foreground">
          Search for a codebase
        </h2>
        <p className="text-lg text-muted-foreground">
          Select an indexed project to begin your deep research analysis.
        </p>
      </div>

      <div className="space-y-8">
        <div className="relative group">
          <SearchIcon className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground group-focus-within:text-primary transition-colors" />
          <input
            type="text"
            placeholder="Filter by name or path..."
            value={codebaseFilter}
            onChange={(e) => setCodebaseFilter(e.target.value)}
            className="w-full bg-card border border-border/50 rounded-2xl pl-12 pr-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 focus:border-primary/50 transition-all shadow-sm text-lg"
          />
        </div>

        <div className="space-y-1">
          {filteredCodebases.length > 0 ? (
            filteredCodebases.map((cb) => (
              <div
                key={cb.id}
                className="group flex items-center justify-between p-6 rounded-2xl hover:bg-muted/30 transition-all cursor-default"
              >
                <div className="space-y-1.5 flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <Database className="h-4 w-4 text-primary" />
                    <h3 className="text-xl font-bold tracking-tight truncate">
                      {cb.name}
                    </h3>
                  </div>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Folder className="h-3.5 w-3.5" />
                    <code className="truncate font-mono bg-muted/50 px-1.5 rounded text-xs">
                      {cb.path}
                    </code>
                  </div>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground/70">
                    <Clock className="h-3.5 w-3.5" />
                    <span>Last indexed: {cb.indexedAt}</span>
                  </div>
                </div>

                <button
                  type="button"
                  onClick={() => handleNewResearch(cb.name)}
                  className="ml-6 flex items-center gap-2 px-5 py-2.5 bg-primary text-primary-foreground rounded-full text-sm font-bold shadow-lg shadow-primary/20 hover:scale-105 active:scale-95 transition-all opacity-0 group-hover:opacity-100"
                >
                  Start Research
                  <ArrowRight className="h-4 w-4" />
                </button>
              </div>
            ))
          ) : (
            <div className="py-20 text-center space-y-3">
              <p className="text-muted-foreground text-lg">
                No codebases match your search.
              </p>
              <button
                type="button"
                onClick={() => handleNewResearch()}
                className="text-primary font-semibold hover:underline"
              >
                Start a general research instead
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

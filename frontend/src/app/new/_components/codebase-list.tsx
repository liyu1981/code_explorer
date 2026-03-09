"use client";

import { useAtom } from "jotai";
import {
  ArrowRight,
  Clock,
  Database,
  Folder,
  Search as SearchIcon,
  Loader2,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import useSWR from "swr";
import { API_URL, fetcher, api } from "@/lib/api";
import {
  createSession,
  researchSessionsAtom,
} from "../../_jotai/research-store";

interface Codebase {
  id: number;
  name: string;
  root_path: string;
  indexed_at: string;
  file_count: number;
  chunk_count: number;
}

export function CodebaseList() {
  const [, setSessions] = useAtom(researchSessionsAtom);
  const [codebaseFilter, setCodebaseFilter] = useState("");
  const [existingSessions, setExistingSessions] = useState<Record<number, any>>(
    {},
  );
  const router = useRouter();

  const {
    data: codebases,
    error,
    isLoading,
  } = useSWR<Codebase[]>(`${API_URL}/api/codemogger/codebases`, fetcher);

  useEffect(() => {
    const loadSessions = async () => {
      try {
        const response = await api.get("/api/research/sessions");
        const sessions = response.data;
        const sessionMap: Record<number, any> = {};
        for (const s of sessions) {
          if (!s.archivedAt) {
            sessionMap[s.codebaseId] = s;
          }
        }
        setExistingSessions(sessionMap);
      } catch (e) {
        console.error("Failed to load sessions", e);
      }
    };
    loadSessions();
  }, []);

  const handleNewResearch = async (cb?: Codebase) => {
    const codebaseId = cb?.id || 0;

    // 1. Create new session
    const newSession = createSession(codebaseId);
    if (cb) {
      newSession.title = `Research: ${cb.name || cb.root_path}`;
    }

    // 2. Save to backend
    await api.post("/api/research/sessions", {
      id: newSession.id,
      codebaseId: newSession.codebaseId,
      title: newSession.title,
      state: newSession.state,
      createdAt: newSession.createdAt,
    });

    setSessions((current) => {
      // Dedup just in case
      const filtered = current.filter((s) => s.id !== newSession.id);
      return [...filtered, newSession];
    });
    router.push(`/research?id=${newSession.id}`);
  };

  const handleContinueResearch = (sessionId: string) => {
    router.push(`/research?id=${sessionId}`);
  };

  const filteredCodebases =
    codebases?.filter(
      (c) =>
        (c.name || "").toLowerCase().includes(codebaseFilter.toLowerCase()) ||
        (c.root_path || "")
          .toLowerCase()
          .includes(codebaseFilter.toLowerCase()),
    ) || [];

  if (error) {
    return (
      <div className="max-w-3xl mx-auto py-20 text-center space-y-4">
        <p className="text-destructive text-lg font-medium">
          Failed to load codebases
        </p>
        <button
          type="button"
          onClick={() => handleNewResearch()}
          className="text-primary font-semibold hover:underline"
        >
          Start a general research instead
        </button>
      </div>
    );
  }

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
          {isLoading ? (
            <div className="py-20 flex flex-col items-center justify-center gap-4 text-muted-foreground">
              <Loader2 className="h-8 w-8 animate-spin" />
              <p>Loading codebases...</p>
            </div>
          ) : filteredCodebases.length > 0 ? (
            filteredCodebases.map((cb) => (
              <div
                key={cb.id}
                className="group flex items-center justify-between p-6 rounded-2xl hover:bg-muted/30 transition-all cursor-default"
              >
                <div className="space-y-1.5 flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <Database className="h-4 w-4 text-primary" />
                    <h3 className="text-xl font-bold tracking-tight truncate">
                      {cb.name || "Unnamed Codebase"}
                    </h3>
                  </div>
                  <div className="flex items-center gap-2 text-sm text-muted-foreground">
                    <Folder className="h-3.5 w-3.5" />
                    <code className="truncate font-mono bg-muted/50 px-1.5 rounded text-xs">
                      {cb.root_path}
                    </code>
                  </div>
                  <div className="flex items-center gap-4 text-xs text-muted-foreground/70">
                    <div className="flex items-center gap-1.5">
                      <Clock className="h-3.5 w-3.5" />
                      <span>
                        Last indexed: {new Date(cb.indexed_at).toLocaleString()}
                      </span>
                    </div>
                    <div className="flex items-center gap-1.5 border-l border-border pl-4">
                      <span>{cb.file_count} files</span>
                      <span>•</span>
                      <span>{cb.chunk_count} chunks</span>
                    </div>
                  </div>
                </div>

                <div className="ml-6 flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                  {existingSessions[cb.id] && (
                    <button
                      type="button"
                      onClick={() =>
                        handleContinueResearch(existingSessions[cb.id].id)
                      }
                      className="flex items-center gap-2 px-5 py-2.5 bg-secondary text-secondary-foreground rounded-full text-sm font-bold shadow-lg shadow-secondary/20 hover:scale-105 active:scale-95 transition-all"
                    >
                      Continue
                      <ArrowRight className="h-4 w-4" />
                    </button>
                  )}
                  <button
                    type="button"
                    onClick={() => handleNewResearch(cb)}
                    className="flex items-center gap-2 px-5 py-2.5 bg-primary text-primary-foreground rounded-full text-sm font-bold shadow-lg shadow-primary/20 hover:scale-105 active:scale-95 transition-all"
                  >
                    New Research
                    <ArrowRight className="h-4 w-4" />
                  </button>
                </div>
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

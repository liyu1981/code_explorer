"use client";

import { useAtom } from "jotai";
import {
  ArrowRight,
  Clock,
  Database,
  Folder,
  Search as SearchIcon,
  Loader2,
  Plus,
  X,
  RefreshCw,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import useSWR from "swr";
import { API_URL, fetcher, api } from "@/lib/api";
import {
  createSession,
  researchSessionsAtom,
} from "../../_jotai/research-store";
import { useWebSocketContext } from "../../_components/websocket-provider";
import { cn } from "@/lib/utils";

interface Codebase {
  id: number;
  name: string;
  rootPath: string;
  indexedAt: number;
  fileCount: number;
  chunkCount: number;
}

interface IndexProgress {
  current: number;
  total: number;
  stage: string;
}

export function CodebaseList() {
  const [, setSessions] = useAtom(researchSessionsAtom);
  const [codebaseFilter, setCodebaseFilter] = useState("");
  const [existingSessions, setExistingSessions] = useState<Record<number, any>>(
    {},
  );
  const [isAddingNew, setIsAddingNew] = useState(false);
  const [newPath, setNewPath] = useState("");
  const [indexingPath, setIndexingPath] = useState<string | null>(null);
  const [indexProgress, setIndexProgress] = useState<IndexProgress | null>(
    null,
  );

  const router = useRouter();
  const { subscribe, unsubscribe } = useWebSocketContext();

  const {
    data: codebases,
    error,
    isLoading,
    mutate,
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

  useEffect(() => {
    const onProgress = (payload: IndexProgress) => {
      setIndexProgress(payload);
    };
    const onDone = (payload: any) => {
      setIndexingPath(null);
      setIndexProgress(null);
      mutate(); // Refresh list
      if (payload.error) {
        alert(`Indexing failed: ${payload.error}`);
      }
    };

    subscribe("index_progress", onProgress);
    subscribe("index_done", onDone);

    return () => {
      unsubscribe("index_progress", onProgress);
      unsubscribe("index_done", onDone);
    };
  }, [subscribe, unsubscribe, mutate]);

  const handleNewResearch = async (cb?: Codebase) => {
    const codebaseId = cb?.id || 0;
    const codebasePath = cb?.rootPath || "";

    // 1. Create new session
    const newSession = createSession(codebaseId, codebasePath);
    if (cb) {
      newSession.title = `Research: ${cb.name || cb.rootPath}`;
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
      const filtered = current.filter((s) => s.id !== newSession.id);
      return [...filtered, newSession];
    });
    router.push(`/research?id=${newSession.id}`);
  };

  const handleCreateIndex = async (path: string) => {
    try {
      setIndexingPath(path);
      setIsAddingNew(false);
      await api.post("/api/codemogger/index", { dir: path });
    } catch (e) {
      console.error("Failed to start indexing", e);
      setIndexingPath(null);
      alert(
        "Failed to start indexing. Make sure the path is absolute and accessible.",
      );
    }
  };

  const handleContinueResearch = (sessionId: string) => {
    router.push(`/research?id=${sessionId}`);
  };

  const filteredCodebases =
    codebases?.filter(
      (c) =>
        (c.name || "").toLowerCase().includes(codebaseFilter.toLowerCase()) ||
        (c.rootPath || "").toLowerCase().includes(codebaseFilter.toLowerCase()),
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
        <div className="flex items-center gap-4">
          <div className="relative group flex-1">
            <SearchIcon className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground group-focus-within:text-primary transition-colors" />
            <input
              type="text"
              placeholder="Filter by name or path..."
              value={codebaseFilter}
              onChange={(e) => setCodebaseFilter(e.target.value)}
              className="w-full bg-card border border-border/50 rounded-2xl pl-12 pr-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 focus:border-primary/50 transition-all shadow-sm text-lg"
            />
          </div>

          <button
            onClick={() => setIsAddingNew(true)}
            className="flex items-center gap-2 px-6 py-4 bg-primary text-primary-foreground rounded-2xl font-bold shadow-lg shadow-primary/20 hover:scale-105 active:scale-95 transition-all whitespace-nowrap"
          >
            <Plus className="h-5 w-5" />
            New Codebase
          </button>
        </div>

        {isAddingNew && (
          <div className="bg-card border border-primary/20 rounded-3xl p-8 shadow-2xl animate-in zoom-in-95 duration-300">
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-xl font-bold">Add Local Codebase</h3>
              <button
                onClick={() => setIsAddingNew(false)}
                className="p-2 hover:bg-muted rounded-full"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                handleCreateIndex(newPath);
              }}
              className="space-y-6"
            >
              <div className="space-y-2">
                <label className="text-xs font-bold text-muted-foreground uppercase tracking-widest px-1">
                  Absolute Directory Path
                </label>
                <input
                  type="text"
                  placeholder="/home/user/project"
                  value={newPath}
                  onChange={(e) => setNewPath(e.target.value)}
                  className="w-full bg-muted/30 border border-border/60 rounded-2xl px-4 py-4 outline-none focus:ring-4 focus:ring-primary/10 transition-all font-mono text-sm"
                  autoFocus
                />
              </div>
              <button
                type="submit"
                className="w-full py-4 bg-primary text-primary-foreground rounded-2xl font-bold shadow-lg shadow-primary/20 hover:scale-[1.02] active:scale-[0.98] transition-all"
              >
                Start Indexing
              </button>
            </form>
          </div>
        )}

        {indexingPath && (
          <div className="bg-primary/5 border border-primary/20 rounded-3xl p-8 space-y-6 animate-pulse">
            <div className="flex items-center gap-4">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
              <div className="flex-1">
                <h3 className="text-lg font-bold">Indexing Codebase...</h3>
                <p className="text-xs text-muted-foreground font-mono truncate">
                  {indexingPath}
                </p>
              </div>
            </div>

            {indexProgress && (
              <div className="space-y-3">
                <div className="flex justify-between text-xs font-bold uppercase tracking-widest text-primary/70">
                  <span>{indexProgress.stage}</span>
                  <span>
                    {indexProgress.current} / {indexProgress.total}
                  </span>
                </div>
                <div className="h-2 w-full bg-primary/10 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-primary transition-all duration-500 ease-out"
                    style={{
                      width: `${(indexProgress.current / indexProgress.total) * 100}%`,
                    }}
                  />
                </div>
              </div>
            )}
          </div>
        )}

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
                      {cb.rootPath}
                    </code>
                  </div>
                  <div className="flex items-center gap-4 text-xs text-muted-foreground/70">
                    <div className="flex items-center gap-1.5">
                      <Clock className="h-3.5 w-3.5" />
                      <span>
                        Last indexed:{" "}
                        {cb.indexedAt
                          ? new Date(cb.indexedAt * 1000).toLocaleString()
                          : "Never"}
                      </span>
                    </div>
                    <div className="flex items-center gap-1.5 border-l border-border pl-4">
                      <span>{cb.fileCount} files</span>
                      <span>•</span>
                      <span>{cb.chunkCount} chunks</span>
                    </div>
                  </div>
                </div>

                <div className="ml-6 flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    type="button"
                    onClick={() => handleCreateIndex(cb.rootPath)}
                    className="p-2.5 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-full transition-all"
                    title="Refresh Index"
                    disabled={!!indexingPath}
                  >
                    <RefreshCw
                      className={cn(
                        "h-5 w-5",
                        indexingPath === cb.rootPath && "animate-spin",
                      )}
                    />
                  </button>

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

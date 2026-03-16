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
  Tag,
  GitBranch,
  History,
  Brain,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import useSWR from "swr";
import { API_URL, fetcher, api } from "@/lib/api";
import {
  createSession,
  researchSessionsAtom,
  activeSessionIdAtom,
} from "../../_jotai/research-store";
import { useWebSocketContext } from "../../_components/websocket-provider";
import { cn } from "@/lib/utils";
import * as Dialog from "@radix-ui/react-dialog";

interface Codebase {
  id: string;
  name: string;
  rootPath: string;
  type: string;
  version: string;
  createdAt: number;
}

interface IndexingStatus {
  status: "not_indexed" | "indexed";
  indexedAt?: number;
  fileCount?: number;
  chunkCount?: number;
}

interface IndexProgress {
  current: number;
  total: number;
  stage: string;
}

function ResearchDialog({
  isOpen,
  onClose,
  sessions,
  onSelectSession,
  onNewResearch,
}: {
  isOpen: boolean;
  onClose: () => void;
  sessions: any[];
  onSelectSession: (sessionId: string) => void;
  onNewResearch: () => void;
}) {
  const [searchQuery, setSearchQuery] = useState("");

  const filteredSessions = sessions
    .filter(
      (s) =>
        s.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
        s.codebaseName?.toLowerCase().includes(searchQuery.toLowerCase()),
    )
    .slice(0, 6);

  return (
    <Dialog.Root open={isOpen} onOpenChange={onClose}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 animate-in fade-in duration-300" />
        <Dialog.Content className="fixed top-[40%] left-1/2 -translate-x-1/2 -translate-y-1/2 w-full max-w-lg max-h-[80vh] bg-background border border-border rounded-2xl shadow-2xl z-50 p-0 animate-in zoom-in-95 duration-300 overflow-hidden">
          <div className="p-6 pb-4">
            <div className="flex items-center justify-between mb-4">
              <Dialog.Title className="text-xl font-bold tracking-tight">
                Research
              </Dialog.Title>
              <Dialog.Close className="p-2 hover:bg-muted rounded-full transition-colors">
                <X className="h-5 w-5" />
              </Dialog.Close>
            </div>

            <div className="relative">
              <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground/50" />
              <input
                type="text"
                placeholder="Search research sessions..."
                className="w-full bg-muted/50 border-none rounded-xl py-2.5 pl-10 pr-4 text-sm focus:ring-2 focus:ring-primary/20 transition-all outline-none"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
          </div>

          <div className="px-6 max-h-[420px] overflow-auto pr-2">
            {filteredSessions.length > 0 ? (
              <div className="space-y-2">
                {filteredSessions.map((s) => (
                  <button
                    key={s.id}
                    type="button"
                    onClick={() => onSelectSession(s.id)}
                    className="w-full text-left p-3 rounded-xl border border-border hover:border-primary/60 hover:bg-muted/50 transition-all group"
                  >
                    <div className="flex items-center justify-between mb-1">
                      <h4 className="font-bold text-sm text-foreground transition-colors truncate max-w-[200px]">
                        {s.title}
                      </h4>
                      <span className="text-[10px] text-muted-foreground font-mono">
                        {new Date(s.createdAt).toLocaleDateString()}
                      </span>
                    </div>
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <span className="truncate">{s.codebaseName}</span>
                      <span>•</span>
                      <span>
                        {s.state === "reported" ? "Complete" : "In Progress"}
                      </span>
                    </div>
                  </button>
                ))}
              </div>
            ) : (
              <div className="text-center py-8 text-muted-foreground text-sm">
                {searchQuery
                  ? "No sessions match your search"
                  : "No research sessions yet"}
              </div>
            )}
          </div>

          <div className="p-4 mt-2 border-t border-border bg-muted/20">
            <button
              type="button"
              onClick={onNewResearch}
              className="w-full flex items-center justify-center gap-2 px-5 py-3 bg-primary text-primary-foreground rounded-xl text-sm font-bold shadow-lg shadow-primary/20 hover:scale-[1.02] active:scale-98 transition-all"
            >
              <Plus className="h-4 w-4" />
              Start New Research
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function CodebaseItem({
  cb,
  sessions,
  indexingPath,
  handleCreateIndex,
  handleBuildKnowledge,
  handleResearch,
}: {
  cb: Codebase;
  sessions: any[];
  indexingPath: string | null;
  handleCreateIndex: (path: string) => void;
  handleBuildKnowledge: (cb: Codebase) => void;
  handleResearch: (cb: Codebase, sessions: any[]) => void;
}) {
  const { data: status, isLoading } = useSWR<IndexingStatus>(
    `${API_URL}/api/codemogger/status?codebase_id=${cb.id}`,
    fetcher,
  );

  return (
    <div className="group flex items-center justify-between p-6 rounded-2xl hover:bg-muted/30 transition-all cursor-default">
      <div className="space-y-1.5 flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <Database className="h-4 w-4 text-primary" />
          <h3 className="text-xl font-bold tracking-tight truncate">
            {cb.name || "Unnamed Codebase"}
          </h3>
          {cb.type === "local" && (
            <span className="px-1.5 py-0.5 rounded-md bg-muted text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
              Local
            </span>
          )}
        </div>
        <div className="flex items-center gap-4 text-sm text-muted-foreground">
          <div className="flex items-center gap-1.5 min-w-0">
            <Folder className="h-3.5 w-3.5" />
            <code className="truncate font-mono bg-muted/50 px-1.5 rounded text-xs">
              {cb.rootPath}
            </code>
          </div>
          {cb.version && (
            <div className="flex items-center gap-1.5 border-l border-border pl-4">
              <GitBranch className="h-3.5 w-3.5" />
              <span className="font-mono text-xs">{cb.version}</span>
            </div>
          )}
        </div>

        <div className="flex items-center gap-4 text-xs text-muted-foreground/70">
          {isLoading ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : status?.status === "indexed" ? (
            <>
              <div className="flex items-center gap-1.5">
                <Clock className="h-3.5 w-3.5" />
                <span>
                  Last indexed:{" "}
                  {status.indexedAt
                    ? new Date(status.indexedAt * 1000).toLocaleString()
                    : "Never"}
                </span>
              </div>
              <div className="flex items-center gap-1.5 border-l border-border pl-4">
                <span>{status.fileCount} files</span>
                <span>•</span>
                <span>{status.chunkCount} chunks</span>
              </div>
            </>
          ) : (
            <span className="text-warning font-medium">Not indexed yet</span>
          )}
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

        <button
          type="button"
          onClick={() => handleBuildKnowledge(cb)}
          className="p-2.5 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-full transition-all"
          title="Initialize Knowledge"
        >
          <Brain className="h-5 w-5" />
        </button>

        <button
          type="button"
          onClick={() => handleResearch(cb, sessions)}
          className="flex items-center gap-2 px-5 py-2.5 bg-primary text-primary-foreground rounded-full text-sm font-bold shadow-lg shadow-primary/20 hover:scale-105 active:scale-95 transition-all"
        >
          Research
          <ArrowRight className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}

export function CodebaseList() {
  const [, setSessions] = useAtom(researchSessionsAtom);
  const [, setActiveSessionId] = useAtom(activeSessionIdAtom);
  const [codebaseFilter, setCodebaseFilter] = useState("");
  const [existingSessions, setExistingSessions] = useState<
    Record<string, any[]>
  >({});
  const [isAddingNew, setIsAddingNew] = useState(false);
  const [newPath, setNewPath] = useState("");
  const [indexingPath, setIndexingPath] = useState<string | null>(null);
  const [indexProgress, setIndexProgress] = useState<IndexProgress | null>(
    null,
  );

  // Dialog state
  const [isResearchDialogOpen, setIsResearchDialogOpen] = useState(false);
  const [dialogCodebase, setDialogCodebase] = useState<Codebase | null>(null);
  const [dialogSessions, setDialogSessions] = useState<any[]>([]);

  const router = useRouter();
  const { subscribe, unsubscribe } = useWebSocketContext();

  const {
    data: codebases,
    error,
    isLoading,
    mutate,
  } = useSWR<Codebase[]>(`${API_URL}/api/codebases`, fetcher);

  useEffect(() => {
    const loadSessions = async () => {
      try {
        const response = await api.get(
          "/api/research/sessions?includeArchived=true",
        );
        const sessions = response.data;
        const sessionMap: Record<string, any[]> = {};
        for (const s of sessions) {
          if (!sessionMap[s.codebaseId]) {
            sessionMap[s.codebaseId] = [];
          }
          sessionMap[s.codebaseId].push(s);
        }
        setExistingSessions(sessionMap);
      } catch (e) {
        console.error("Failed to load sessions", e);
      }
    };
    loadSessions();
  }, []);

  useEffect(() => {
    const onProgress = (payload: any) => {
      setIndexProgress(payload);
    };

    const onFinished = (payload: any) => {
      setIndexingPath(null);
      setIndexProgress(null);
      mutate();
    };

    subscribe("codemogger.index.progress", onProgress);
    subscribe("codemogger.index.finished", onFinished);

    return () => {
      unsubscribe("codemogger.index.progress", onProgress);
      unsubscribe("codemogger.index.finished", onFinished);
    };
  }, [subscribe, unsubscribe, mutate]);

  const handleCreateIndex = async (path: string) => {
    setIndexingPath(path);
    try {
      await api.post("/api/codemogger/index", { dir: path });
    } catch (e) {
      console.error("Indexing failed", e);
      setIndexingPath(null);
    }
  };

  const handleAddNewCodebase = async () => {
    if (!newPath) return;
    try {
      await api.post("/api/codemogger/index", { dir: newPath });
      setIsAddingNew(false);
      setNewPath("");
      mutate();
    } catch (e) {
      console.error("Failed to add codebase", e);
    }
  };

  const handleNewResearch = async (cb: Codebase) => {
    const newSession = createSession(cb.id, cb.rootPath, cb.name, cb.version);
    setSessions((prev) => [newSession, ...prev]);
    setActiveSessionId(newSession.id);

    try {
      await api.post("/api/research/sessions", {
        id: newSession.id,
        codebaseId: newSession.codebaseId,
        title: newSession.title,
        state: newSession.state,
        createdAt: newSession.createdAt,
      });
      router.push(`/research?id=${newSession.id}`);
    } catch (e) {
      console.error("Failed to save new session", e);
    }
  };

  const handleBuildKnowledge = async (cb: Codebase) => {
    router.push(`/knowledge?cbid=${cb.id}`);
  };

  const handleResearch = (cb: Codebase, sessions: any[]) => {
    setDialogCodebase(cb);
    setDialogSessions(sessions);
    setIsResearchDialogOpen(true);
  };

  const selectSession = async (sessionId: string) => {
    setIsResearchDialogOpen(false);

    // Check if session is already in memory
    setSessions((prev) => {
      const existing = prev.find((s) => s.id === sessionId);
      if (existing) {
        setActiveSessionId(sessionId);
        router.push(`/research?id=${sessionId}`);
        return prev;
      }

      // If not in memory, it will be rehydrated in the research page
      // but we need a placeholder to avoid empty state flash or just push
      setActiveSessionId(sessionId);
      router.push(`/research?id=${sessionId}`);
      return prev;
    });
  };

  const handleNewResearchFromDialog = async () => {
    if (!dialogCodebase) return;
    setIsResearchDialogOpen(false);
    await handleNewResearch(dialogCodebase);
  };

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[400px] gap-4">
        <Loader2 className="h-10 w-10 animate-spin text-primary/50" />
        <p className="text-muted-foreground animate-pulse">
          Loading your codebases...
        </p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[400px] text-destructive gap-4">
        <X className="h-10 w-10" />
        <p>Failed to load codebases. Is the backend running?</p>
      </div>
    );
  }

  const filteredCodebases = codebases?.filter(
    (cb) =>
      cb.name.toLowerCase().includes(codebaseFilter.toLowerCase()) ||
      cb.rootPath.toLowerCase().includes(codebaseFilter.toLowerCase()),
  );

  return (
    <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <ResearchDialog
        isOpen={isResearchDialogOpen}
        onClose={() => setIsResearchDialogOpen(false)}
        sessions={dialogSessions}
        onSelectSession={selectSession}
        onNewResearch={handleNewResearchFromDialog}
      />

      <div className="flex items-center justify-between">
        <div className="relative flex-1 max-w-xl">
          <SearchIcon className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground/50" />
          <input
            type="text"
            placeholder="Search codebases by name or path..."
            className="w-full bg-muted/50 border-none rounded-2xl py-4 pl-12 pr-4 text-lg focus:ring-4 focus:ring-primary/10 transition-all outline-none"
            value={codebaseFilter}
            onChange={(e) => setCodebaseFilter(e.target.value)}
          />
        </div>
        <button
          type="button"
          onClick={() => setIsAddingNew(true)}
          className="flex items-center gap-2 bg-primary text-primary-foreground px-6 py-4 rounded-2xl font-bold shadow-xl shadow-primary/20 hover:scale-105 active:scale-95 transition-all"
        >
          <Plus className="h-5 w-5" />
          Add Codebase
        </button>
      </div>

      {isAddingNew && (
        <div className="bg-card border border-primary/20 rounded-3xl p-8 shadow-2xl animate-in zoom-in-95 duration-300">
          <div className="flex items-center justify-between mb-6">
            <h3 className="text-2xl font-bold tracking-tight">
              Add New Codebase
            </h3>
            <button
              type="button"
              onClick={() => setIsAddingNew(false)}
              className="p-2 hover:bg-muted rounded-full transition-colors"
            >
              <X className="h-6 w-6" />
            </button>
          </div>
          <div className="space-y-4">
            <div className="space-y-2">
              <label
                htmlFor="path"
                className="text-sm font-bold text-muted-foreground uppercase tracking-widest px-1"
              >
                Local Directory Path
              </label>
              <input
                id="path"
                type="text"
                placeholder="/absolute/path/to/project"
                className="w-full bg-muted/50 border-border rounded-xl p-4 font-mono text-sm focus:ring-2 focus:ring-primary outline-none"
                value={newPath}
                onChange={(e) => setNewPath(e.target.value)}
              />
            </div>
            <div className="flex justify-end gap-3 pt-4">
              <button
                type="button"
                onClick={() => setIsAddingNew(false)}
                className="px-6 py-3 rounded-xl font-bold hover:bg-muted transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleAddNewCodebase}
                className="bg-primary text-primary-foreground px-8 py-3 rounded-xl font-bold shadow-lg shadow-primary/20 hover:scale-105 active:scale-95 transition-all"
              >
                Initialize Index
              </button>
            </div>
          </div>
        </div>
      )}

      {indexingPath && indexProgress && (
        <div className="bg-primary/5 border border-primary/20 rounded-2xl p-6 animate-pulse">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-3">
              <Loader2 className="h-5 w-5 animate-spin text-primary" />
              <span className="font-bold text-primary">
                Indexing codebase...
              </span>
            </div>
            <span className="text-sm font-mono text-primary/70">
              {Math.round((indexProgress.current / indexProgress.total) * 100)}%
            </span>
          </div>
          <div className="w-full bg-primary/10 rounded-full h-2 overflow-hidden">
            <div
              className="bg-primary h-full transition-all duration-300"
              style={{
                width: `${(indexProgress.current / indexProgress.total) * 100}%`,
              }}
            />
          </div>
          <p className="mt-2 text-xs text-primary/60 font-medium">
            Stage: {indexProgress.stage} ({indexProgress.current} /{" "}
            {indexProgress.total})
          </p>
        </div>
      )}

      <div className="grid gap-4">
        {filteredCodebases?.map((cb) => (
          <CodebaseItem
            key={cb.id}
            cb={cb}
            sessions={existingSessions[cb.id] || []}
            indexingPath={indexingPath}
            handleCreateIndex={handleCreateIndex}
            handleBuildKnowledge={handleBuildKnowledge}
            handleResearch={handleResearch}
          />
        ))}
        {filteredCodebases?.length === 0 && (
          <div className="text-center py-24 bg-muted/20 rounded-3xl border border-dashed border-border">
            <Database className="h-16 w-16 text-muted-foreground/20 mx-auto mb-4" />
            <h3 className="text-xl font-bold text-muted-foreground">
              No codebases found
            </h3>
            <p className="text-muted-foreground/60">
              Try a different search or add a new codebase to get started.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}

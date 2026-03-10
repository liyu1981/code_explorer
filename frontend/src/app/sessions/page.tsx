"use client";

import { useAtom } from "jotai";
import {
  Archive,
  Database,
  ExternalLink,
  Folder,
  Loader2,
  Search,
  Trash2,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import useSWR from "swr";
import { API_URL, api, fetcher } from "@/lib/api";
import { cn } from "@/lib/utils";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import { researchSessionsAtom } from "../_jotai/research-store";

interface Session {
  id: string;
  codebaseId: string;
  codebasePath: string;
  codebaseName: string;
  codebaseVersion: string;
  title: string;
  state: string;
  createdAt: number;
  archivedAt?: number;
}

interface Codebase {
  id: string;
  name: string;
}

export default function SessionsManagementPage() {
  const [page, setPage] = useState(1);
  const [pageSize] = useState(10);
  const [codebaseFilter, setCodebaseFilter] = useState("");
  const [, setSessions] = useAtom(researchSessionsAtom);
  const router = useRouter();

  const { data: codebases } = useSWR<Codebase[]>(
    `${API_URL}/api/codebases`,
    fetcher,
  );

  const { data, error, isLoading, mutate } = useSWR(
    `${API_URL}/api/research/sessions/manage?page=${page}&pageSize=${pageSize}&codebaseId=${codebaseFilter}`,
    fetcher,
  );

  const sessions = data?.sessions as Session[];
  const total = data?.total || 0;
  const totalPages = Math.ceil(total / pageSize);

  const handleArchive = async (id: string) => {
    try {
      await api.post(`/api/research/sessions/${id}/archive`);
      toast.success("Session archived successfully");
      mutate();
    } catch (e) {
      console.error("Archive failed", e);
      toast.error("Failed to archive session");
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this session?")) return;
    try {
      await api.delete(`/api/research/sessions/${id}`);
      toast.success("Session deleted successfully");
      mutate();
      // Also remove from local state if present
      setSessions((prev) => prev.filter((s) => s.id !== id));
    } catch (e) {
      console.error("Delete failed", e);
      toast.error("Failed to delete session");
    }
  };

  const handleContinue = (session: Session) => {
    // Add to local sessions if not there
    setSessions((prev) => {
      if (prev.find((s) => s.id === session.id)) return prev;
      return [
        ...prev,
        {
          id: session.id,
          codebaseId: session.codebaseId,
          codebasePath: session.codebasePath,
          codebaseName: session.codebaseName,
          codebaseVersion: session.codebaseVersion,
          title: session.title,
          state: session.state as any,
          createdAt: session.createdAt,
          archivedAt: session.archivedAt,
          turns: [],
          steps: [],
          thoughtProcess: "",
        },
      ];
    });
    router.push(`/research?id=${session.id}`);
  };

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center justify-between w-full">
          <h1 className="text-xl font-bold tracking-tight text-primary">
            Manage Sessions
          </h1>
        </div>
      </AppHeader>

      <div className="flex-1 overflow-auto p-6 bg-background/50">
        <div className="max-w-6xl mx-auto w-full">
          <div className="flex items-center justify-between mb-6">
            <p className="text-muted-foreground text-sm">
              Manage and organize your research sessions across different
              codebases.
            </p>
            <div className="relative w-64">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground/50" />
              <select
                className="w-full bg-card border border-border/60 rounded-xl py-2 pl-9 pr-4 text-sm focus:ring-4 focus:ring-primary/10 transition-all outline-none appearance-none cursor-pointer font-medium"
                value={codebaseFilter}
                onChange={(e) => {
                  setCodebaseFilter(e.target.value);
                  setPage(1);
                }}
              >
                <option value="">All Codebases</option>
                {codebases?.map((cb) => (
                  <option key={cb.id} value={cb.id}>
                    {cb.name}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {isLoading ? (
            <div className="flex items-center justify-center py-24">
              <Loader2 className="h-8 w-8 animate-spin text-primary/50" />
            </div>
          ) : error ? (
            <div className="text-center py-24 text-destructive">
              Failed to load sessions.
            </div>
          ) : sessions?.length === 0 ? (
            <div className="text-center py-24 bg-muted/20 rounded-3xl border border-dashed border-border">
              <p className="text-muted-foreground">No sessions found.</p>
            </div>
          ) : (
            <div className="bg-card border border-border rounded-2xl overflow-hidden shadow-sm">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="bg-muted/30 border-b border-border">
                    <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
                      Title / Codebase
                    </th>
                    <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
                      Status
                    </th>
                    <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
                      Created At
                    </th>
                    <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest text-right">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border/50">
                  {sessions.map((session) => (
                    <tr
                      key={session.id}
                      className="hover:bg-muted/10 transition-colors"
                    >
                      <td className="px-6 py-4">
                        <div className="flex flex-col gap-1">
                          <span className="font-bold text-foreground">
                            {session.title}
                          </span>
                          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                            <Folder className="h-3 w-3" />
                            <span>{session.codebaseName}</span>
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-2">
                          {session.archivedAt ? (
                            <span className="px-2 py-0.5 rounded-full bg-muted text-[10px] font-bold uppercase tracking-wider text-muted-foreground border border-border/50">
                              Archived
                            </span>
                          ) : (
                            <span className="px-2 py-0.5 rounded-full bg-primary/10 text-[10px] font-bold uppercase tracking-wider text-primary border border-primary/20">
                              Active
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="px-6 py-4 text-sm text-muted-foreground font-mono">
                        {new Date(session.createdAt).toLocaleString()}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <button
                            type="button"
                            onClick={() => handleContinue(session)}
                            className="p-2 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-lg transition-all"
                            title="Continue Research"
                          >
                            <ExternalLink className="h-4 w-4" />
                          </button>
                          {!session.archivedAt && (
                            <button
                              type="button"
                              onClick={() => handleArchive(session.id)}
                              className="p-2 text-muted-foreground hover:text-orange-500 hover:bg-orange-500/10 rounded-lg transition-all"
                              title="Archive"
                            >
                              <Archive className="h-4 w-4" />
                            </button>
                          )}
                          <button
                            type="button"
                            onClick={() => handleDelete(session.id)}
                            className="p-2 text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-lg transition-all"
                            title="Delete"
                          >
                            <Trash2 className="h-4 w-4" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>

              {totalPages > 1 && (
                <div className="px-6 py-4 border-t border-border bg-muted/20 flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">
                    Showing {(page - 1) * pageSize + 1} to{" "}
                    {Math.min(page * pageSize, total)} of {total} sessions
                  </span>
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      disabled={page === 1}
                      onClick={() => setPage((p) => p - 1)}
                      className="px-4 py-1.5 rounded-lg border border-border bg-background text-sm font-medium disabled:opacity-50 hover:bg-muted transition-colors"
                    >
                      Previous
                    </button>
                    <button
                      type="button"
                      disabled={page === totalPages}
                      onClick={() => setPage((p) => p + 1)}
                      className="px-4 py-1.5 rounded-lg border border-border bg-background text-sm font-medium disabled:opacity-50 hover:bg-muted transition-colors"
                    >
                      Next
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </AppContainer>
  );
}

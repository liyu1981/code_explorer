"use client";

import {
  Bookmark,
  ExternalLink,
  Folder,
  Loader2,
  Search,
  Trash2,
} from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { toast } from "sonner";
import useSWR from "swr";
import { API_URL, api, fetcher } from "@/lib/api";
import { cn } from "@/lib/utils";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";

interface SavedReport {
  id: string;
  sessionId: string;
  codebaseId: string;
  title: string;
  query: string;
  content: string;
  codebaseName: string;
  codebasePath: string;
  createdAt: number;
}

export default function SavedReportsPage() {
  const [page, setPage] = useState(1);
  const [pageSize] = useState(10);
  const [searchQuery, setSearchQuery] = useState("");
  const router = useRouter();

  const { data, error, isLoading, mutate } = useSWR(
    `${API_URL}/api/saved_reports?page=${page}&pageSize=${pageSize}&q=${encodeURIComponent(searchQuery)}`,
    fetcher,
  );

  const reports = data?.reports as SavedReport[];
  const total = data?.total || 0;
  const totalPages = Math.ceil(total / pageSize);

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this saved report?")) return;
    try {
      await api.delete(`/api/saved_reports/${id}`);
      toast.success("Snapshot deleted successfully");
      mutate();
    } catch (e) {
      console.error("Delete failed", e);
      toast.error("Failed to delete snapshot");
    }
  };

  const handleOpen = (id: string) => {
    router.push(`/saved_report?id=${id}`);
  };

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center gap-4 w-full">
          <Bookmark className="h-5 w-5 text-primary" />
          <h1 className="text-xl font-bold tracking-tight text-primary">
            Saved Reports
          </h1>
        </div>
      </AppHeader>

      <div className="flex-1 overflow-auto p-6 bg-background/50">
        <div className="max-w-6xl mx-auto w-full">
          <div className="flex items-center justify-between mb-6">
            <p className="text-muted-foreground text-sm">
              Search and browse through your saved research snapshots.
            </p>
            <div className="relative w-80">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground/50" />
              <input
                type="text"
                placeholder="Search snapshots..."
                className="w-full bg-card border border-border/60 rounded-xl py-2 pl-9 pr-4 text-sm focus:ring-4 focus:ring-primary/10 transition-all outline-none"
                value={searchQuery}
                onChange={(e) => {
                  setSearchQuery(e.target.value);
                  setPage(1);
                }}
              />
            </div>
          </div>

          {isLoading ? (
            <div className="flex items-center justify-center py-24">
              <Loader2 className="h-8 w-8 animate-spin text-primary/50" />
            </div>
          ) : error ? (
            <div className="text-center py-24 text-destructive">
              Failed to load saved reports.
            </div>
          ) : reports?.length === 0 ? (
            <div className="text-center py-24 bg-muted/20 rounded-3xl border border-dashed border-border">
              <Bookmark className="h-12 w-12 text-muted-foreground/20 mx-auto mb-4" />
              <p className="text-muted-foreground">No saved reports found.</p>
            </div>
          ) : (
            <div className="bg-card border border-border rounded-2xl overflow-hidden shadow-sm">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="bg-muted/30 border-b border-border">
                    <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
                      Report Snapshot
                    </th>
                    <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
                      Codebase
                    </th>
                    <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest">
                      Saved At
                    </th>
                    <th className="px-6 py-4 text-xs font-bold text-muted-foreground uppercase tracking-widest text-right">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border/50">
                  {reports.map((report) => (
                    <tr
                      key={report.id}
                      className="hover:bg-muted/10 transition-colors"
                    >
                      <td className="px-6 py-4">
                        <div className="flex flex-col gap-1 max-w-md">
                          <span className="font-bold text-foreground truncate">
                            {report.query}
                          </span>
                          <span className="text-xs text-muted-foreground truncate italic">
                            Part of: {report.title}
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
                          <Folder className="h-3.5 w-3.5" />
                          <span>{report.codebaseName}</span>
                        </div>
                      </td>
                      <td className="px-6 py-4 text-sm text-muted-foreground font-mono">
                        {new Date(report.createdAt).toLocaleDateString()}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <button
                            type="button"
                            onClick={() => handleOpen(report.id)}
                            className="p-2 text-muted-foreground hover:text-primary hover:bg-primary/10 rounded-lg transition-all"
                            title="Open Snapshot"
                          >
                            <ExternalLink className="h-4 w-4" />
                          </button>
                          <button
                            type="button"
                            onClick={() => handleDelete(report.id)}
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
                    {Math.min(page * pageSize, total)} of {total} snapshots
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

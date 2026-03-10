"use client";
import { Bookmark, Folder, Loader2, Trash2, X } from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect } from "react";
import { useAtom } from "jotai";
import { toast } from "sonner";
import useSWR from "swr";
import { API_URL, api, fetcher } from "@/lib/api";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import { Markdown } from "../_components/markdown";
import { activeSavedReportsAtom } from "../_jotai/ui-store";

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

function SavedReportContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const id = searchParams.get("id");
  const [, setActiveReports] = useAtom(activeSavedReportsAtom);

  const {
    data: report,
    error,
    isLoading,
  } = useSWR<SavedReport>(
    id ? `${API_URL}/api/saved_reports/${id}` : null,
    fetcher,
  );

  useEffect(() => {
    if (report) {
      setActiveReports((prev) => {
        if (prev.find((r) => r.id === report.id)) return prev;
        return [
          ...prev,
          { id: report.id, title: report.title, query: report.query },
        ];
      });
    }
  }, [report, setActiveReports]);

  const handleDelete = async () => {
    if (!id || !confirm("Are you sure you want to delete this snapshot?"))
      return;
    try {
      await api.delete(`/api/saved_reports/${id}`);
      setActiveReports((prev) => prev.filter((r) => r.id !== id));
      toast.success("Snapshot deleted successfully");
      router.push("/saved_reports");
    } catch (e) {
      console.error("Delete failed", e);
      toast.error("Failed to delete snapshot");
    }
  };

  const handleClose = () => {
    if (id) {
      setActiveReports((prev) => prev.filter((r) => r.id !== id));
    }
    router.push("/saved_reports");
  };

  if (isLoading) {
    return (
      <AppContainer>
        <div className="flex-1 flex items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-primary/50" />
        </div>
      </AppContainer>
    );
  }

  if (error || !report) {
    return (
      <AppContainer>
        <div className="flex-1 flex flex-col items-center justify-center gap-4">
          <p className="text-destructive font-bold">Failed to load snapshot.</p>
          <button
            onClick={handleClose}
            className="px-4 py-2 bg-muted rounded-lg hover:bg-muted/80 transition-colors"
          >
            Go Back
          </button>
        </div>
      </AppContainer>
    );
  }

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center gap-4 w-full">
          <div className="flex items-center gap-3">
            <Bookmark className="h-5 w-5 text-primary" />
            <h1 className="text-xl font-bold tracking-tight text-primary">
              Saved Snapshot
            </h1>
            <div className="h-5 w-px bg-border/60 mx-1" />
            <div className="flex items-center gap-2 px-3 py-1.5 bg-muted/40 rounded-lg border border-border/40">
              <Folder className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-xs font-mono text-muted-foreground truncate max-w-[400px]">
                {report.codebaseName}
              </span>
            </div>
          </div>
          <div className="flex items-center gap-2 ml-auto">
            <button
              type="button"
              onClick={handleClose}
              className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-muted-foreground hover:text-foreground hover:bg-muted rounded-md transition-colors"
              title="Close Snapshot"
            >
              <X className="h-4 w-4" />
              Close
            </button>
            <button
              type="button"
              onClick={handleDelete}
              className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-md transition-colors"
              title="Delete Snapshot"
            >
              <Trash2 className="h-4 w-4" />
              Delete
            </button>
          </div>
        </div>
      </AppHeader>

      <div className="flex-1 overflow-auto bg-background/50">
        <div className="max-w-4xl mx-auto w-full py-12 px-6">
          <div className="space-y-8">
            <div className="space-y-4">
              <h2 className="text-3xl font-bold tracking-tight text-foreground">
                {report.query}
              </h2>
              <div className="flex items-center gap-4 text-xs text-muted-foreground font-medium uppercase tracking-widest">
                <span>Session: {report.title}</span>
                <span>•</span>
                <span>
                  Saved on {new Date(report.createdAt).toLocaleString()}
                </span>
              </div>
            </div>

            <div className="prose prose-invert max-w-none">
              <Markdown content={report.content} />
            </div>
          </div>
        </div>
      </div>
    </AppContainer>
  );
}

export default function SavedReportPage() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <SavedReportContent />
    </Suspense>
  );
}

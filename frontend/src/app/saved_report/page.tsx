"use client";
import { Loader2, X } from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useMemo } from "react";
import { useAtom } from "jotai";
import useSWR from "swr";
import { API_URL, api, fetcher } from "@/lib/api";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import { LoadingState } from "../_components/loading-state";
import { ErrorState } from "../_components/error-state";
import { activeSavedReportsAtom } from "../_jotai/ui-store";
import { ResearchReport } from "../research/_components/research-report";
import type { ResearchTurn } from "../_jotai/research-store";
import type { Source } from "../research/_components/source-card";

interface SavedReport {
  id: string;
  sessionId: string;
  codebaseId: string;
  title: string;
  query: string;
  streamData: string;
  codebaseName: string;
  codebasePath: string;
  createdAt: number;
}

interface OpenAIChunk {
  choices: {
    delta: {
      content?: string;
    };
  }[];
}

interface CEEvent {
  object: string;
  id?: string;
  status?: "pending" | "active" | "completed";
  label?: string;
  content?: string;
  source?: Source;
  resource?: Source;
  query?: string;
  timestamp?: number;
}

function parseStreamData(streamData: string): ResearchTurn[] {
  const lines = streamData.split("\n\n");
  const turns: ResearchTurn[] = [];
  let currentTurnId = "";

  for (const line of lines) {
    if (!line.trim()) continue;

    if (line.startsWith("data: ")) {
      const data = line.slice(6);
      if (data === "[DONE]") continue;
      try {
        const chunk: OpenAIChunk = JSON.parse(data);
        const content = chunk.choices[0]?.delta?.content;
        if (content && currentTurnId) {
          const turn = turns.find((t) => t.id === currentTurnId);
          if (turn) {
            turn.report += content;
          }
        }
      } catch (e) {}
    } else if (line.startsWith("ce: ")) {
      const data = line.slice(4);
      try {
        const ce: CEEvent = JSON.parse(data);
        switch (ce.object) {
          case "research.turn.started":
            currentTurnId = ce.id!;
            turns.push({
              id: currentTurnId,
              query: ce.query!,
              report: "",
              sources: [],
              timestamp: ce.timestamp!,
            });
            break;
          case "research.source.added":
            if (ce.source && currentTurnId) {
              const turn = turns.find((t) => t.id === currentTurnId);
              if (turn) {
                turn.sources.push(ce.source);
              }
            }
            break;
          case "resource.material":
            if (ce.resource && currentTurnId) {
              const turn = turns.find((t) => t.id === currentTurnId);
              if (turn) {
                turn.sources.push(ce.resource);
              }
            }
            break;
        }
      } catch (e) {}
    }
  }

  return turns;
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

  const turns = useMemo(() => {
    if (!report?.streamData) return [];
    return parseStreamData(report.streamData);
  }, [report?.streamData]);

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

  const handleClose = () => {
    if (id) {
      setActiveReports((prev) => prev.filter((r) => r.id !== id));
    }
    router.push("/saved_reports");
  };

  if (isLoading) {
    return (
      <AppContainer>
        <LoadingState className="flex-1" />
      </AppContainer>
    );
  }

  if (error || !report) {
    return (
      <AppContainer>
        <div className="flex-1 p-6">
          <ErrorState
            title="Failed to load snapshot"
            message="The snapshot may have been deleted or moved."
          />
          <div className="flex justify-center mt-6">
            <button
              onClick={handleClose}
              className="px-4 py-2 bg-muted rounded-lg hover:bg-muted/80 transition-colors font-bold"
            >
              Go Back
            </button>
          </div>
        </div>
      </AppContainer>
    );
  }

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center gap-4 w-full">
          <div className="flex items-center gap-3">
            <span className="px-2 py-1 text-[10px] font-bold text-muted-foreground bg-muted rounded-md uppercase tracking-widest">
              CODEBASE: {report.codebaseName}
            </span>
            <h1 className="text-xl font-bold tracking-tight text-primary truncate max-w-[600px]">
              {report.title}
            </h1>
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
          </div>
        </div>
      </AppHeader>

      <div className="flex-1 overflow-auto bg-background/50">
        <div className="max-w-6xl mx-auto w-full py-8 px-6">
          {turns.length > 0 ? (
            <ResearchReport turns={turns} hideTurnInfo={true} />
          ) : (
            <div className="text-center text-muted-foreground py-12">
              No research data available
            </div>
          )}
        </div>
      </div>
    </AppContainer>
  );
}

export default function SavedReportPage() {
  return (
    <Suspense
      fallback={
        <AppContainer>
          <LoadingState className="flex-1" />
        </AppContainer>
      }
    >
      <SavedReportContent />
    </Suspense>
  );
}

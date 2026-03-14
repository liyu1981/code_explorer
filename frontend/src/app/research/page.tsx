"use client";

import { useAtom } from "jotai";
import { Archive, Folder, X, Search } from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef } from "react";
import { toast } from "sonner";
import { API_URL, api } from "@/lib/api";
import { cn } from "@/lib/utils";
import { useWebSocketContext } from "../_components/websocket-provider";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import {
  activeSessionIdAtom,
  researchSessionsAtom,
  type ResearchSession,
  type ResearchTurn,
} from "../_jotai/research-store";
import { FloatingThoughtProcess } from "./_components/floating-thought-process";
import { IdleResearchView } from "./_components/idle-research-view";
import { ReasoningTrace } from "./_components/reasoning-trace";
import { ResearchInput } from "./_components/research-input";
import { ResearchReport } from "./_components/research-report";
import { StickyResearchInput } from "./_components/sticky-research-input";
import type { Source } from "./_components/source-card";

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

function ResearchContent() {
  const [sessions, setSessions] = useAtom(researchSessionsAtom);
  const [activeSessionId, setActiveSessionId] = useAtom(activeSessionIdAtom);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const rehydratingRef = useRef<string | null>(null);
  const { subscribe, unsubscribe } = useWebSocketContext();

  const router = useRouter();
  const searchParams = useSearchParams();
  const urlId = searchParams.get("id");

  // Research Session Updates
  useEffect(() => {
    const handleResearchUpdate = (payload: any) => {
      if (payload.type === "research.session.updated") {
        const { sessionId, title } = payload;
        setSessions((prev) =>
          prev.map((s) => (s.id === sessionId ? { ...s, title } : s)),
        );
      }
    };

    subscribe("research", handleResearchUpdate);
    return () => unsubscribe("research", handleResearchUpdate);
  }, [subscribe, unsubscribe, setSessions]);

  // Rehydration Effect
  useEffect(() => {
    const rehydrate = async () => {
      if (!urlId || rehydratingRef.current === urlId) return;

      // If already fully in memory, skip
      const existing = sessions.find((s) => s.id === urlId);
      if (existing && existing.turns.length > 0) return;

      rehydratingRef.current = urlId;
      try {
        const sessResponse = await api.get(
          "/api/research/sessions?includeArchived=true",
        );
        const allSessions = sessResponse.data;
        const sessionData = allSessions.find((s: any) => s.id === urlId);

        if (!sessionData) {
          rehydratingRef.current = null;
          return;
        }

        const reportsResponse = await api.get(
          `/api/research/sessions/${urlId}/reports`,
        );
        const reports = reportsResponse.data || [];

        // Reconstruct session state from events
        const session: ResearchSession = {
          id: sessionData.id,
          codebaseId: sessionData.codebaseId,
          codebasePath: sessionData.codebasePath,
          codebaseName: sessionData.codebaseName,
          codebaseVersion: sessionData.codebaseVersion,
          title: sessionData.title,
          state: sessionData.state as any,
          createdAt: sessionData.createdAt,
          archivedAt: sessionData.archivedAt,
          steps: [],
          thoughtProcess: "",
          turns: [],
        };

        // Simple replayer for each report
        for (const report of reports) {
          const lines = report.streamData.split("\n\n");
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
                  const turn = session.turns.find(
                    (t) => t.id === currentTurnId,
                  );
                  if (turn) {
                    turn.report += content;
                    turn.updatedAt = report.updatedAt;
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
                    session.turns.push({
                      id: currentTurnId,
                      query: ce.query!,
                      report: "",
                      sources: [],
                      timestamp: ce.timestamp!,
                      updatedAt: report.updatedAt,
                    });
                    break;
                  case "research.step.update": {
                    const idx = session.steps.findIndex((s) => s.id === ce.id);
                    if (idx > -1) {
                      session.steps[idx] = {
                        ...session.steps[idx],
                        status: ce.status!,
                        label: ce.label!,
                      };
                    } else if (ce.id && ce.label && ce.status) {
                      session.steps.push({
                        id: ce.id,
                        label: ce.label,
                        status: ce.status,
                      });
                    }
                    break;
                  }
                  case "research.reasoning.delta":
                    session.thoughtProcess += ce.content || "";
                    break;
                  case "research.source.added":
                    if (ce.source && currentTurnId) {
                      const turn = session.turns.find(
                        (t) => t.id === currentTurnId,
                      );
                      if (turn) turn.sources.push(ce.source);
                    }
                    break;
                  case "resource.material":
                    if (ce.resource && currentTurnId) {
                      const turn = session.turns.find(
                        (t) => t.id === currentTurnId,
                      );
                      if (turn) turn.sources.push(ce.resource);
                    }
                    break;
                }
              } catch (e) {}
            }
          }
        }

        setSessions((prev) => {
          const filtered = prev.filter((s) => s.id !== session.id);
          return [...filtered, session];
        });
      } catch (e) {
        console.error("Rehydration failed", e);
      } finally {
        rehydratingRef.current = null;
      }
    };

    rehydrate();
  }, [urlId, setSessions]); // Remove sessions from deps to avoid loop

  // Sync activeSessionId with URL
  useEffect(() => {
    if (urlId && urlId !== activeSessionId) {
      setActiveSessionId(urlId);
    } else if (!urlId) {
      router.push("/");
    }
  }, [urlId, activeSessionId, setActiveSessionId, router]);

  const activeSession = sessions.find((s) => s.id === activeSessionId);
  const prevTurnsLengthRef = useRef(0);

  // biome-ignore lint/correctness/useExhaustiveDependencies: scroll management
  useEffect(() => {
    if (!scrollContainerRef.current) return;
    const turnsLength = activeSession?.turns.length ?? 0;
    const activeTurnId = activeSession?.activeTurnId;
    const isStreaming = !!activeTurnId;
    const isNewTurn = turnsLength > prevTurnsLengthRef.current;

    // Update ref for next run
    if (isNewTurn) {
      prevTurnsLengthRef.current = turnsLength;
    }

    if (turnsLength > 1) {
      const currentTurnId = isStreaming
        ? activeTurnId
        : activeSession?.turns[turnsLength - 1]?.id;
      const turnElement = scrollContainerRef.current.querySelector(
        `[data-turn-id="${currentTurnId}"]`,
      );

      if (turnElement) {
        const offset = 16;
        const targetTop = (turnElement as HTMLElement).offsetTop - offset;
        const currentTop = scrollContainerRef.current.scrollTop;

        // If we are significantly off target (> 5px), and we are either in a "new turn" event
        // OR we are currently streaming that turn, retry the scroll.
        if (Math.abs(currentTop - targetTop) > 5) {
          scrollContainerRef.current.scrollTo({
            top: targetTop,
            behavior: isNewTurn ? "smooth" : "auto", // Smooth for first jump, auto for micro-adjustments
          });
        }
      }
    } else if (activeSession?.state === "searching") {
      // During initial reasoning of the VERY FIRST turn, scroll to bottom to see logs
      setTimeout(() => {
        scrollContainerRef.current?.scrollTo({
          top: scrollContainerRef.current.scrollHeight,
          behavior: "smooth",
        });
      }, 100);
    }

    // We explicitly omit report/thoughtProcess length from deps to "stop moving" during streaming
  }, [
    activeSession?.turns.length,
    activeSession?.activeTurnId,
    activeSession?.state === "searching",
    // This dependency ensures we "retry" as new content comes in, in case of layout shifts
    activeSession?.turns[activeSession?.turns.length - 1]?.report.length,
  ]);

  const handleSearch = async (
    sessionId: string,
    query: string,
    _deep: boolean,
  ) => {
    // We don't initialize turn here manually anymore,
    // it will be initialized by "research.turn.started" event from backend
    setSessions((current) =>
      current.map((s) =>
        s.id === sessionId
          ? {
              ...s,
              state: "searching",
              thoughtProcess: "",
              steps: [], // Clear steps for the new turn
            }
          : s,
      ),
    );

    try {
      const response = await fetch(`${API_URL}/api/agent/research`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ query, sessionId }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const reader = response.body?.getReader();
      if (!reader) throw new Error("No reader");

      const decoder = new TextDecoder();
      let buffer = "";
      let currentTurnId = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n\n");
        buffer = lines.pop() || "";

        for (const line of lines) {
          if (line.startsWith("data: ")) {
            const data = line.slice(6);
            if (data === "[DONE]") break;

            try {
              const chunk: OpenAIChunk = JSON.parse(data);
              const content = chunk.choices[0]?.delta?.content;
              if (content && currentTurnId) {
                setSessions((current) =>
                  current.map((s) =>
                    s.id === sessionId
                      ? {
                          ...s,
                          turns: s.turns.map((t) =>
                            t.id === currentTurnId
                              ? {
                                  ...t,
                                  report: t.report + content,
                                  updatedAt: Date.now(),
                                }
                              : t,
                          ),
                        }
                      : s,
                  ),
                );
              }
            } catch (e) {
              console.error("Failed to parse data chunk", e, data);
            }
          } else if (line.startsWith("ce: ")) {
            const data = line.slice(4);
            try {
              const event: CEEvent = JSON.parse(data);
              setSessions((current) =>
                current.map((s) => {
                  if (s.id !== sessionId) return s;

                  switch (event.object) {
                    case "research.turn.started":
                      currentTurnId = event.id!;
                      return {
                        ...s,
                        activeTurnId: currentTurnId,
                        turns: [
                          ...s.turns,
                          {
                            id: currentTurnId,
                            query: event.query!,
                            report: "",
                            sources: [],
                            timestamp: event.timestamp!,
                            updatedAt: Date.now(),
                          },
                        ],
                      };
                    case "research.step.update": {
                      const updatedSteps = [...s.steps];
                      const existingStepIdx = updatedSteps.findIndex(
                        (st) => st.id === event.id,
                      );

                      if (existingStepIdx > -1) {
                        updatedSteps[existingStepIdx] = {
                          ...updatedSteps[existingStepIdx],
                          status:
                            event.status ??
                            updatedSteps[existingStepIdx].status,
                          label:
                            event.label ?? updatedSteps[existingStepIdx].label,
                        };
                      } else if (event.id && event.label && event.status) {
                        updatedSteps.push({
                          id: event.id,
                          label: event.label,
                          status: event.status,
                        });
                      }

                      return {
                        ...s,
                        steps: updatedSteps,
                      };
                    }
                    case "research.reasoning.delta":
                      return {
                        ...s,
                        thoughtProcess:
                          s.thoughtProcess + (event.content ?? ""),
                      };
                    case "research.source.added":
                      if (event.source) {
                        return {
                          ...s,
                          turns: s.turns.map((t) =>
                            t.id === currentTurnId
                              ? { ...t, sources: [...t.sources, event.source!] }
                              : t,
                          ),
                        };
                      }
                      return s;
                    case "resource.material":
                      if (event.resource) {
                        return {
                          ...s,
                          turns: s.turns.map((t) =>
                            t.id === currentTurnId
                              ? {
                                  ...t,
                                  sources: [...t.sources, event.resource!],
                                }
                              : t,
                          ),
                        };
                      }
                      return s;
                    default:
                      return s;
                  }
                }),
              );
            } catch (e) {
              console.error("Failed to parse CE event", e, data);
            }
          }
        }
      }
    } catch (error) {
      console.error("Research failed:", error);
    } finally {
      // Finalize in memory
      setSessions((current) => {
        const updated = current.map((s) =>
          s.id === sessionId
            ? {
                ...s,
                state: "reported" as const,
                activeTurnId: undefined,
              }
            : s,
        );

        // Persist final session state to backend
        const session = updated.find((s) => s.id === sessionId);
        if (session) {
          api
            .post("/api/research/sessions", {
              id: session.id,
              codebaseId: session.codebaseId,
              title: session.title,
              state: "reported",
              createdAt: session.createdAt,
              archivedAt: session.archivedAt,
            })
            .then(() => {
              // If it's the first turn, summarize it to get a proper title
              if (session.turns.length === 1) {
                api
                  .post(`/api/research/sessions/${session.id}/summarize`)
                  .then((res) => {
                    const updatedSess = res.data;
                    setSessions((prev) =>
                      prev.map((s) =>
                        s.id === updatedSess.id
                          ? { ...s, title: updatedSess.title }
                          : s,
                      ),
                    );
                  });
              }
            });
        }

        return updated;
      });
    }
  };

  const handleArchive = async (id: string) => {
    try {
      await api.post(`/api/research/sessions/${id}/archive`);
    } catch (e) {
      console.error("Archive failed", e);
    }

    setSessions((current) => current.filter((s) => s.id !== id));
    router.push("/new");
  };

  const handleClose = (id: string) => {
    setSessions((current) => current.filter((s) => s.id !== id));
    router.push("/new");
  };

  const handleDeleteTurn = async (turnId: string) => {
    if (!activeSessionId) return;
    try {
      await api.delete(
        `/api/research/sessions/${activeSessionId}/reports/${turnId}`,
      );
      setSessions((current) =>
        current.map((s) =>
          s.id === activeSessionId
            ? { ...s, turns: s.turns.filter((t) => t.id !== turnId) }
            : s,
        ),
      );
    } catch (e) {
      console.error("Delete turn failed", e);
    }
  };

  const handleRegenerate = async (turn: ResearchTurn) => {
    if (!activeSessionId) return;
    await handleDeleteTurn(turn.id);
    handleSearch(activeSessionId, turn.query, false);
  };

  const handleSaveTurn = async (turn: ResearchTurn) => {
    if (!activeSession) return;
    try {
      await api.post("/api/saved_reports", {
        sessionId: activeSession.id,
        codebaseId: activeSession.codebaseId,
        title: activeSession.title,
        query: turn.query,
        turnId: turn.id,
        codebaseName: activeSession.codebaseName,
        codebasePath: activeSession.codebasePath,
      });
      toast.success("Snapshot saved successfully!");
    } catch (e) {
      console.error("Save snapshot failed", e);
      toast.error("Failed to save snapshot.");
    }
  };

  if (!activeSession) {
    return null;
  }

  const isResearching =
    activeSession.state === "searching" || activeSession.state === "reasoning";

  const isIdle =
    activeSession.state === "idle" && activeSession.turns.length === 0;

  const followUpSuggestions = [
    "Analyze performance implications",
    "How is this tested?",
    "Are there security concerns?",
  ];

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center gap-4 w-full">
          <div className="flex items-center gap-3">
            <Search className="h-5 w-5 text-primary" />
            <h1 className="text-xl font-bold tracking-tight text-primary truncate max-w-[600px]">
              {activeSession.title}
            </h1>
          </div>
          <div className="flex items-center gap-2 ml-auto">
            <button
              type="button"
              onClick={() => handleClose(activeSession.id)}
              className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-muted-foreground hover:text-foreground hover:bg-muted rounded-md transition-colors"
              title="Close Page"
            >
              <X className="h-4 w-4" />
              Close
            </button>
            <button
              type="button"
              onClick={() => handleArchive(activeSession.id)}
              className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-md transition-colors"
              title="Archive Research"
            >
              <Archive className="h-4 w-4" />
              Archive
            </button>
          </div>
        </div>
      </AppHeader>

      <div className="flex-1 flex flex-col relative overflow-hidden">
        {/* Content Area */}
        <div
          ref={scrollContainerRef}
          className={cn(
            "flex-1 overflow-auto transition-all duration-500",
            isIdle ? "flex items-center justify-center" : "p-6",
          )}
        >
          {isIdle ? (
            <IdleResearchView
              onSearch={(q, deep) => handleSearch(activeSession.id, q, deep)}
            />
          ) : (
            (activeSession.turns.length > 0 || isResearching) && (
              <div className="max-w-6xl mx-auto w-full space-y-12 pb-48">
                <ResearchReport
                  turns={activeSession.turns}
                  onDeleteTurn={handleDeleteTurn}
                  onRegenerateTurn={handleRegenerate}
                  onSaveTurn={handleSaveTurn}
                  isStreaming={isResearching}
                />
                <div className="h-[400px]" />
              </div>
            )
          )}
        </div>

        {/* Floating Thought Process Indicator */}
        <FloatingThoughtProcess
          isVisible={isResearching}
          steps={activeSession.steps}
          thoughtProcess={activeSession.thoughtProcess}
        />

        {/* Sticky Input Area */}
        <StickyResearchInput
          isVisible={!isIdle}
          onSearch={(q, deep) => handleSearch(activeSession.id, q, deep)}
          suggestions={!isResearching ? followUpSuggestions : []}
        />
      </div>
    </AppContainer>
  );
}

export default function ResearchPage() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <ResearchContent />
    </Suspense>
  );
}

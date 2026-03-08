"use client";

import { useAtom } from "jotai";
import { Archive } from "lucide-react";
import { nanoid } from "nanoid";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef } from "react";
import { API_URL } from "@/lib/api";
import { cn } from "@/lib/utils";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import {
  activeSessionIdAtom,
  researchSessionsAtom,
} from "../_jotai/research-store";
import { ReasoningTrace } from "./_components/reasoning-trace";
import { ResearchInput } from "./_components/research-input";
import { ResearchReport } from "./_components/research-report";
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
  content?: string;
  source?: Source;
  resource?: Source;
}

function ResearchContent() {
  const [sessions, setSessions] = useAtom(researchSessionsAtom);
  const [activeSessionId, setActiveSessionId] = useAtom(activeSessionIdAtom);
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  const router = useRouter();
  const searchParams = useSearchParams();
  const urlId = searchParams.get("id");

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
    const turnId = nanoid();

    // Initialize turn in session
    setSessions((current) =>
      current.map((s) =>
        s.id === sessionId
          ? {
              ...s,
              state: "searching",
              activeTurnId: turnId,
              thoughtProcess: "",
              turns: [
                ...s.turns,
                {
                  id: turnId,
                  query,
                  report: "",
                  sources: [],
                  timestamp: Date.now(),
                },
              ],
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
        body: JSON.stringify({ query }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const reader = response.body?.getReader();
      if (!reader) throw new Error("No reader");

      const decoder = new TextDecoder();
      let buffer = "";

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
              if (content) {
                setSessions((current) =>
                  current.map((s) =>
                    s.id === sessionId
                      ? {
                          ...s,
                          turns: s.turns.map((t) =>
                            t.id === turnId
                              ? { ...t, report: t.report + content }
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
                    case "research.step.update":
                      return {
                        ...s,
                        steps: s.steps.map((step) =>
                          step.id === event.id
                            ? { ...step, status: event.status ?? step.status }
                            : step,
                        ),
                      };
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
                            t.id === turnId
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
                            t.id === turnId
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
      // Finalize
      setSessions((current) =>
        current.map((s) =>
          s.id === sessionId
            ? {
                ...s,
                state: "reported",
                activeTurnId: undefined,
              }
            : s,
        ),
      );
    }
  };

  const handleArchive = (id: string) => {
    setSessions((current) => current.filter((s) => s.id !== id));
    router.push("/");
  };

  if (!activeSession) {
    return null;
  }

  const isResearching =
    activeSession.state === "searching" || activeSession.state === "reasoning";

  return (
    <AppContainer>
      <AppHeader>
        <div className="flex items-center gap-4 w-full">
          <h1 className="text-xl font-bold tracking-tight text-primary">
            Research
          </h1>
          <button
            onClick={() => handleArchive(activeSession.id)}
            className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-md transition-colors ml-auto"
            title="Archive Research"
          >
            <Archive className="h-4 w-4" />
            Archive
          </button>
        </div>
      </AppHeader>

      <div className="flex-1 flex flex-col relative overflow-hidden">
        {/* Content Area */}
        <div
          ref={scrollContainerRef}
          className={cn(
            "flex-1 overflow-auto transition-all duration-500",
            activeSession.state === "idle"
              ? "flex items-center justify-center"
              : "p-6",
          )}
        >
          {activeSession.state === "idle" && (
            <div className="w-full max-w-4xl space-y-12 animate-in fade-in zoom-in-95 duration-700">
              <div className="text-center space-y-4">
                <h2 className="text-6xl font-bold tracking-tighter">
                  What are we building?
                </h2>
                <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
                  Research your codebase with semantic intelligence and deep
                  analytical reasoning.
                </p>
              </div>
              <div className="max-w-3xl mx-auto p-8">
                <ResearchInput
                  onSearch={(q, deep) =>
                    handleSearch(activeSession.id, q, deep)
                  }
                />
              </div>
            </div>
          )}

          {(activeSession.turns.length > 0 || isResearching) && (
            <div className="max-w-6xl mx-auto w-full space-y-12 pb-48">
              <ResearchReport
                turns={activeSession.turns}
                onFollowUp={(q) => handleSearch(activeSession.id, q, true)}
                isStreaming={isResearching}
              />
            </div>
          )}
        </div>

        {/* Floating Thought Process Indicator */}
        {isResearching && (
          <div className="absolute top-[1rem] left-0 right-0 z-50 px-6 animate-in fade-in slide-in-from-top-4 duration-500 pointer-events-none">
            <div className="max-w-3xl mx-auto pointer-events-auto">
              <div className="bg-background/80 backdrop-blur-xl border border-primary/20 shadow-2xl rounded-2xl overflow-hidden shadow-primary/5">
                <div className="p-4 border-b border-border/50 bg-muted/30">
                  <ReasoningTrace steps={activeSession.steps} />
                </div>

                {activeSession.thoughtProcess && (
                  <div className="max-h-[200px] overflow-auto p-4 bg-muted/10">
                    <h4 className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest mb-2 px-1">
                      Granular Thought Process
                    </h4>
                    <pre className="text-xs font-mono whitespace-pre-wrap text-muted-foreground/70 leading-relaxed">
                      {activeSession.thoughtProcess}
                    </pre>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}
        {/* Sticky Input Area */}

        {activeSession.state !== "idle" && (
          <div className="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-background via-background to-transparent pt-20 pb-8 px-6 z-20 pointer-events-none">
            <div className="max-w-3xl mx-auto pointer-events-auto">
              <ResearchInput
                onSearch={(q, deep) => handleSearch(activeSession.id, q, deep)}
                isCompact
              />
            </div>
          </div>
        )}
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

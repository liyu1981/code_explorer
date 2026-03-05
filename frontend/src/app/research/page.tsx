"use client";

import { useAtom } from "jotai";
import { Archive } from "lucide-react";
import { nanoid } from "nanoid";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef } from "react";
import { cn } from "@/lib/utils";
import { AppContainer } from "../_components/app-container";
import { AppHeader } from "../_components/app-header";
import {
  activeSessionIdAtom,
  ResearchTurn,
  researchSessionsAtom,
} from "../_jotai/research-store";
import { ReasoningTrace } from "./_components/reasoning-trace";
import { ResearchInput } from "./_components/research-input";
import { ResearchReport } from "./_components/research-report";
import { Source } from "./_components/source-card";

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

  // Auto scroll to bottom when turns update
  useEffect(() => {
    if (scrollContainerRef.current) {
      setTimeout(() => {
        scrollContainerRef.current?.scrollTo({
          top: scrollContainerRef.current.scrollHeight,
          behavior: "smooth",
        });
      }, 100);
    }
  }, []);

  const handleSearch = (id: string, q: string, _deep: boolean) => {
    setSessions((current) =>
      current.map((s) =>
        s.id === id
          ? {
              ...s,
              state: "searching",
              steps: [
                {
                  id: "1",
                  label: "Indexing codebase context",
                  status: "active",
                },
                {
                  id: "2",
                  label: "Performing semantic search",
                  status: "pending",
                },
                {
                  id: "3",
                  label: "Generating technical synthesis",
                  status: "pending",
                },
              ],
            }
          : s,
      ),
    );

    // Simulate process
    setTimeout(() => {
      setSessions((current) =>
        current.map((s) =>
          s.id === id
            ? {
                ...s,
                steps: s.steps.map((step) =>
                  step.id === "1"
                    ? { ...step, status: "completed" as const }
                    : step.id === "2"
                      ? { ...step, status: "active" as const }
                      : step,
                ),
              }
            : s,
        ),
      );

      setTimeout(() => {
        setSessions((current) =>
          current.map((s) =>
            s.id === id
              ? {
                  ...s,
                  steps: s.steps.map((step) =>
                    step.id === "2"
                      ? { ...step, status: "completed" as const }
                      : step.id === "3"
                        ? { ...step, status: "active" as const }
                        : step,
                  ),
                }
              : s,
          ),
        );

        setTimeout(() => {
          const isFirstTurn = (activeSession?.turns.length || 0) === 0;

          let report = "";
          let sources: Source[] = [];

          if (isFirstTurn) {
            report = `I have analyzed the codebase regarding **${q}**. The project is a sophisticated TypeScript-based React application utilizing Next.js 16.

### Core Component Pattern
The application uses a modular pattern for its UI components. For instance, the WebSocket management is handled via a custom React hook pattern to ensure state synchronization across the app:

\`\`\`typescript
import { useState, useEffect, useCallback } from 'react';

interface WebSocketHook {
  isConnected: boolean;
  lastMessage: any;
  send: (msg: string) => void;
}

export function useSocket(url: string): WebSocketHook {
  const [socket, setSocket] = useState<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    const ws = new WebSocket(url);
    ws.onopen = () => setIsConnected(true);
    ws.onclose = () => setIsConnected(false);
    setSocket(ws);
    return () => ws.close();
  }, [url]);

  const send = useCallback((msg: string) => {
    socket?.send(msg);
  }, [socket]);

  return { isConnected, lastMessage: null, send };
}
\`\`\`

### State Management
State is managed using **Jotai**, which provides atomic state updates. This is particularly useful for the research session history, where each turn is appended to a global state atom.`;

            sources = [
              {
                id: "1",
                path: "src/hooks/useSocket.ts",
                snippet:
                  "export function useSocket(url: string): WebSocketHook {\n  const [socket, setSocket] = useState<WebSocket | null>(null);",
              },
              {
                id: "2",
                path: "src/store/research.ts",
                snippet:
                  "export const researchSessionsAtom = atom<ResearchSession[]>([]);",
              },
            ];
          } else {
            report = `Deepening the analysis on **${q}**, I have examined the Markdown rendering implementation.

### Technical Implementation
The system uses \`react-markdown\` combined with \`rehype-highlight\` for syntax highlighting. The implementation details can be found in the core components:

\`\`\`typescript
import ReactMarkdown from "react-markdown";
import rehypeHighlight from "rehype-highlight";

interface MarkdownProps {
  content: string;
  className?: string;
}

export const MarkdownRenderer: React.FC<MarkdownProps> = ({ content, className }) => {
  return (
    <div className={className}>
      <ReactMarkdown 
        rehypePlugins={[rehypeHighlight]}
        components={{
          code({ node, inline, className, children, ...props }) {
            return !inline ? (
              <pre className="rounded-lg bg-gray-100 p-4">
                <code className={className} {...props}>
                  {children}
                </code>
              </pre>
            ) : (
              <code className="bg-gray-200 px-1 rounded" {...props}>
                {children}
              </code>
            );
          }
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
};
\`\`\`

### Styling Strategy
The project adopts **Tailwind CSS 4** for styling, leveraging the new \`@theme\` configuration for a more robust design system.`;

            sources = [
              {
                id: "3",
                path: "src/components/Markdown.tsx",
                snippet:
                  "export const MarkdownRenderer: React.FC<MarkdownProps> = ({ content, className }) => {",
              },
              {
                id: "4",
                path: "tailwind.config.ts",
                snippet: "@theme {\n  --color-primary: oklch(0.205 0 0);\n}",
              },
            ];
          }

          const newTurn: ResearchTurn = {
            id: nanoid(),
            query: q,
            report,
            sources,
            timestamp: Date.now(),
          };

          setSessions((current) =>
            current.map((s) =>
              s.id === id
                ? {
                    ...s,
                    state: "reported",
                    turns: [...s.turns, newTurn],
                    steps: s.steps.map((step) => ({
                      ...step,
                      status: "completed" as const,
                    })),
                  }
                : s,
            ),
          );
        }, 1500);
      }, 1200);
    }, 1000);
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
              />

              {isResearching && (
                <div className="max-w-3xl mx-auto py-8">
                  <ReasoningTrace steps={activeSession.steps} />
                </div>
              )}
            </div>
          )}
        </div>

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

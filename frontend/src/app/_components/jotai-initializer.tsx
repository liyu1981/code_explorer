"use client";

import { useAtom } from "jotai";
import { useEffect } from "react";
import { api } from "@/lib/api";
import {
  researchSessionsAtom,
  type ResearchSession,
} from "../_jotai/research-store";

export function JotaiInitializer({ children }: { children: React.ReactNode }) {
  const [sessions, setSessions] = useAtom(researchSessionsAtom);

  useEffect(() => {
    const init = async () => {
      try {
        const response = await api.get("/api/research/sessions");
        const allSessions = response.data;

        // Merge backend sessions with existing local ones, ensuring no duplicates by ID
        const activeSessions: ResearchSession[] = allSessions
          .filter((s: any) => !s.archivedAt)
          .map((s: any) => {
            const existing = sessions.find((es) => es.id === s.id);
            return {
              id: s.id,
              codebaseId: s.codebaseId,
              title: s.title,
              state: s.state as any,
              createdAt: s.createdAt,
              archivedAt: s.archivedAt,
              // If we already have turns/steps in memory (from local storage or current session), keep them
              steps: existing?.steps || [],
              thoughtProcess: existing?.thoughtProcess || "",
              turns: existing?.turns || [],
            };
          });

        setSessions(activeSessions);
      } catch (e) {
        console.error("Failed to initialize sessions", e);
      }
    };

    init();
  }, [setSessions]);

  return <>{children}</>;
}

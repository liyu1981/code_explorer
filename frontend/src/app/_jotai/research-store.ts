import { atom } from "jotai";
import { nanoid } from "nanoid";
import { ReasoningStep } from "../research/_components/reasoning-trace";
import { Source } from "../research/_components/source-card";

export type ResearchState = "idle" | "searching" | "reasoning" | "reported";

export interface ResearchTurn {
  id: string;
  query: string;
  report: string;
  sources: Source[];
  timestamp: number;
}

export interface ResearchSession {
  id: string;
  title: string;
  state: ResearchState;
  steps: ReasoningStep[];
  turns: ResearchTurn[];
  createdAt: number;
}

export const researchSessionsAtom = atom<ResearchSession[]>([]);
export const activeSessionIdAtom = atom<string | null>(null);

// Helper to create a new session
export const createSession = (): ResearchSession => ({
  id: nanoid(10),
  title: "New Research",
  state: "idle",
  steps: [
    { id: "1", label: "Searching codebase for context", status: "pending" },
    { id: "2", label: "Analyzing retrieved code chunks", status: "pending" },
    { id: "3", label: "Synthesizing deep research report", status: "pending" },
  ],
  turns: [],
  createdAt: Date.now(),
});

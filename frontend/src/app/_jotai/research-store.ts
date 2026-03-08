import { atom } from "jotai";
import { nanoid } from "nanoid";
import type { ReasoningStep } from "../research/_components/reasoning-trace";
import type { Source } from "../research/_components/source-card";

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
  thoughtProcess: string; // Granular log of reasoning
  turns: ResearchTurn[];
  activeTurnId?: string; // Currently streaming turn ID
  createdAt: number;
}

export const researchSessionsAtom = atom<ResearchSession[]>([]);
export const activeSessionIdAtom = atom<string | null>(null);

// Helper to create a new session
export const createSession = (): ResearchSession => ({
  id: nanoid(10),
  title: "New Research",
  state: "idle",
  steps: [],
  thoughtProcess: "",
  turns: [],
  createdAt: Date.now(),
});

import { atom } from "jotai";
import { atomWithStorage } from "jotai/utils";
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
  updatedAt?: number;
}

export interface ResearchSession {
  id: string;
  codebaseId: string;
  codebasePath: string;
  codebaseName: string;
  codebaseVersion: string;
  title: string;
  state: ResearchState;
  steps: ReasoningStep[];
  thoughtProcess: string; // Granular log of reasoning
  turns: ResearchTurn[];
  activeTurnId?: string; // Currently streaming turn ID
  createdAt: number;
  archivedAt?: number;
}

// Persist research sessions to local storage
export const researchSessionsAtom = atomWithStorage<ResearchSession[]>(
  "ce-research-sessions",
  [],
);
export const activeSessionIdAtom = atom<string | null>(null);

// Helper to create a new session
export const createSession = (
  codebaseId: string,
  codebasePath: string,
  codebaseName: string,
  codebaseVersion: string,
): ResearchSession => ({
  id: nanoid(10),
  codebaseId,
  codebasePath,
  codebaseName,
  codebaseVersion,
  title: "New Research",
  state: "idle",
  steps: [],
  thoughtProcess: "",
  turns: [],
  createdAt: Date.now(),
});

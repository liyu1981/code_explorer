"use client";

import { useRef, useCallback } from "react";
import type {
  ResearchSession,
  ResearchTurn,
} from "@/app/_jotai/research-store";
import type { Source } from "@/app/research/_components/source-card";
import {
  type CEStreamCallbacks,
  processCEStream,
  type CEEvent,
  type OpenAIChunk,
} from "@/lib/cestream";

export type { CEEvent, OpenAIChunk };
export type { ResearchTurn };

function getSourceKey(source: Source): string {
  return `${source.path}::${source.snippet}`;
}

export interface RollbackPoint {
  turnID: string;
  report: string;
  sources: Source[];
}

function createTurnFromEvent(e: CEEvent): ResearchTurn {
  return {
    id: e.id!,
    query: e.query!,
    report: "",
    sources: [],
    timestamp: e.timestamp!,
    updatedAt: Date.now(),
  };
}

function createSourceDedupCallbacks(
  callbacks: CEStreamCallbacks,
  sourceKeysRef: { current: Set<string> },
): CEStreamCallbacks {
  return {
    onOpenaiChunk: callbacks.onOpenaiChunk,
    onLLMTryRunStart: callbacks.onLLMTryRunStart,
    onLLMTryRunEnd: callbacks.onLLMTryRunEnd,
    onLLMTryRunFailed: callbacks.onLLMTryRunFailed,
    onResearchTurnStarted: callbacks.onResearchTurnStarted,
    onResearchStepUpdate: callbacks.onResearchStepUpdate,
    onResearchReasoningDelta: callbacks.onResearchReasoningDelta,
    onResearchSourceAdded: (id, e) => {
      if (e.source) {
        const key = getSourceKey(e.source);
        if (!sourceKeysRef.current.has(key)) {
          sourceKeysRef.current.add(key);
          callbacks.onResearchSourceAdded(id, e);
        }
      }
    },
    onResourceMaterial: (id, e) => {
      if (e.resource) {
        const key = getSourceKey(e.resource);
        if (!sourceKeysRef.current.has(key)) {
          sourceKeysRef.current.add(key);
          callbacks.onResourceMaterial(id, e);
        }
      }
    },
  };
}

type SetSessionsFn = (
  updater: (sessions: ResearchSession[]) => ResearchSession[],
) => void;

export function createStreamingCallbacks(
  sessionId: string,
  setSessions: SetSessionsFn,
  rollbackPointsRef: React.MutableRefObject<Map<string, RollbackPoint[]>>,
): CEStreamCallbacks {
  const sourceKeysRef = { current: new Set<string>() };

  const appendReport = (turnID: string, content: string) => {
    setSessions((current) =>
      current.map((s) =>
        s.id === sessionId
          ? {
              ...s,
              turns: s.turns.map((t) =>
                t.id === turnID
                  ? { ...t, report: t.report + content, updatedAt: Date.now() }
                  : t,
              ),
            }
          : s,
      ),
    );
  };

  const addTurn = (turn: ResearchTurn) => {
    setSessions((current) =>
      current.map((s) =>
        s.id === sessionId
          ? { ...s, activeTurnId: turn.id, turns: [...s.turns, turn] }
          : s,
      ),
    );
  };

  const updateStep = (step: { id: string; label: string; status: string }) => {
    setSessions((current) =>
      current.map((s) => {
        if (s.id !== sessionId) return s;
        const updatedSteps = [...s.steps];
        const existingIdx = updatedSteps.findIndex((st) => st.id === step.id);
        if (existingIdx > -1) {
          updatedSteps[existingIdx] = {
            ...updatedSteps[existingIdx],
            ...step,
            status: step.status as any,
          };
        } else {
          updatedSteps.push({ ...step, status: step.status as any });
        }
        return { ...s, steps: updatedSteps };
      }),
    );
  };

  const appendThoughtProcess = (delta: string) => {
    setSessions((current) =>
      current.map((s) =>
        s.id === sessionId
          ? { ...s, thoughtProcess: s.thoughtProcess + delta }
          : s,
      ),
    );
  };

  const addSource = (turnID: string, source: Source) => {
    const key = getSourceKey(source);
    if (sourceKeysRef.current.has(key)) return;
    sourceKeysRef.current.add(key);

    setSessions((current) =>
      current.map((s) =>
        s.id === sessionId
          ? {
              ...s,
              turns: s.turns.map((t) =>
                t.id === turnID ? { ...t, sources: [...t.sources, source] } : t,
              ),
            }
          : s,
      ),
    );
  };

  const rollback = (turnID: string, report: string, sources: Source[]) => {
    setSessions((current) =>
      current.map((s) =>
        s.id === sessionId
          ? {
              ...s,
              turns: s.turns.map((t) =>
                t.id === turnID ? { ...t, report, sources } : t,
              ),
            }
          : s,
      ),
    );
  };

  const baseCallbacks: CEStreamCallbacks = {
    onOpenaiChunk: (turnID: string, chunk) => {
      const content = chunk.choices[0]?.delta?.content;
      if (content) appendReport(turnID, content);
    },
    onLLMTryRunStart: (turnID: string) => {
      const points = rollbackPointsRef.current.get(sessionId) ?? [];
      points.push({ turnID, report: "", sources: [] });
      rollbackPointsRef.current.set(sessionId, points);
    },
    onLLMTryRunEnd: () => {
      const points = rollbackPointsRef.current.get(sessionId);
      if (points && points.length > 0) points.pop();
    },
    onLLMTryRunFailed: (turnID: string) => {
      const points = rollbackPointsRef.current.get(sessionId);
      if (points && points.length > 0) {
        const lastPoint = points.pop()!;
        rollback(turnID, lastPoint.report, lastPoint.sources);
      }
    },
    onResearchTurnStarted: (_turnID: string, e: CEEvent) => {
      addTurn(createTurnFromEvent(e));
    },
    onResearchStepUpdate: (_: string, e: CEEvent) => {
      if (e.id && e.label && e.status) {
        updateStep({ id: e.id, label: e.label, status: e.status });
      }
    },
    onResearchReasoningDelta: (_: string, e: CEEvent) => {
      appendThoughtProcess(e.content ?? "");
    },
    onResearchSourceAdded: (turnID: string, e: CEEvent) => {
      if (e.source) addSource(turnID, e.source);
    },
    onResourceMaterial: (turnID: string, e: CEEvent) => {
      if (e.resource) addSource(turnID, e.resource);
    },
  };

  return createSourceDedupCallbacks(baseCallbacks, sourceKeysRef);
}

export function createRehydrationCallbacks(
  session: ResearchSession,
): CEStreamCallbacks {
  const sourceKeysRef = { current: new Set<string>() };

  const appendReport = (turnID: string, content: string) => {
    const turn = session.turns.find((t) => t.id === turnID);
    if (turn) turn.report += content;
  };

  const addTurn = (turn: ResearchTurn) => {
    session.turns.push(turn);
  };

  const updateStep = (step: { id: string; label: string; status: string }) => {
    const idx = session.steps.findIndex((s) => s.id === step.id);
    if (idx > -1) {
      session.steps[idx] = {
        ...session.steps[idx],
        ...step,
        status: step.status as any,
      };
    } else {
      session.steps.push({ ...step, status: step.status as any });
    }
  };

  const appendThoughtProcess = (delta: string) => {
    session.thoughtProcess += delta;
  };

  const addSource = (turnID: string, source: Source) => {
    const key = getSourceKey(source);
    if (sourceKeysRef.current.has(key)) return;
    sourceKeysRef.current.add(key);

    const turn = session.turns.find((t) => t.id === turnID);
    if (turn) turn.sources.push(source);
  };

  const baseCallbacks: CEStreamCallbacks = {
    onOpenaiChunk: (turnID: string, chunk) => {
      const content = chunk.choices[0]?.delta?.content;
      if (content) appendReport(turnID, content);
    },
    onLLMTryRunStart: () => {},
    onLLMTryRunEnd: () => {},
    onLLMTryRunFailed: () => {},
    onResearchTurnStarted: (_turnID: string, e: CEEvent) => {
      addTurn(createTurnFromEvent(e));
    },
    onResearchStepUpdate: (_: string, e: CEEvent) => {
      if (e.id && e.label && e.status) {
        updateStep({ id: e.id, label: e.label, status: e.status });
      }
    },
    onResearchReasoningDelta: (_: string, e: CEEvent) => {
      appendThoughtProcess(e.content ?? "");
    },
    onResearchSourceAdded: (turnID: string, e: CEEvent) => {
      if (e.source) addSource(turnID, e.source);
    },
    onResourceMaterial: (turnID: string, e: CEEvent) => {
      if (e.resource) addSource(turnID, e.resource);
    },
  };

  return createSourceDedupCallbacks(baseCallbacks, sourceKeysRef);
}

export function useResearchStreamProcessor() {
  const rollbackPointsRef = useRef<Map<string, RollbackPoint[]>>(new Map());
  const currentTurnIdRef = useRef<string>("");

  const clearRollbackPoints = useCallback((sessionId: string) => {
    rollbackPointsRef.current.delete(sessionId);
  }, []);

  const processResearchStream = useCallback(
    (
      turnID: string,
      reader: ReadableStreamDefaultReader<Uint8Array>,
      callbacks: CEStreamCallbacks,
    ): Promise<void> => {
      return processCEStream(turnID, reader, {
        onOpenaiChunk: callbacks.onOpenaiChunk,
        onLLMTryRunStart: (id, e) => {
          currentTurnIdRef.current = id;
          callbacks.onLLMTryRunStart(id, e);
        },
        onLLMTryRunEnd: callbacks.onLLMTryRunEnd,
        onLLMTryRunFailed: callbacks.onLLMTryRunFailed,
        onResearchTurnStarted: callbacks.onResearchTurnStarted,
        onResearchStepUpdate: callbacks.onResearchStepUpdate,
        onResearchReasoningDelta: callbacks.onResearchReasoningDelta,
        onResearchSourceAdded: callbacks.onResearchSourceAdded,
        onResourceMaterial: callbacks.onResourceMaterial,
      });
    },
    [],
  );

  return {
    rollbackPointsRef,
    currentTurnIdRef,
    clearRollbackPoints,
    processResearchStream,
  };
}

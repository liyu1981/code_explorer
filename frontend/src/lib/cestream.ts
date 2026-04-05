import { createStreamEmitter } from "./stream";

export interface CEStreamSource {
  id: string;
  path: string;
  score?: number;
  snippet?: string;
  start_line?: number;
  end_line?: number;
}

export interface OpenAIChunk {
  choices: {
    delta: {
      content?: string;
    };
  }[];
}

export interface CEEvent {
  turnid: string;
  object: string;
  stepid?: string;
  status?: "pending" | "active" | "completed";
  label?: string;
  content?: string;
  source?: CEStreamSource;
  resource?: CEStreamSource;
  query?: string;
  timestamp?: number;
  tryid?: number;
}

export interface CEStreamCallbacks {
  onOpenaiChunk: (turnID: string, data: OpenAIChunk) => void;
  onLLMTryRunStart: (turnID: string, e: CEEvent) => void;
  onLLMTryRunEnd: (turnID: string, e: CEEvent) => void;
  onLLMTryRunFailed: (turnID: string, e: CEEvent) => void;
  onResearchTurnStarted: (turnID: string, e: CEEvent) => void;
  onResearchStepUpdate: (turnID: string, e: CEEvent) => void;
  onResearchReasoningDelta: (turnID: string, e: CEEvent) => void;
  onResearchSourceAdded: (turnID: string, e: CEEvent) => void;
  onResourceMaterial: (turnID: string, e: CEEvent) => void;
  onToolCallRequest: (turnID: string, e: CEEvent) => void;
  onToolCallResponse: (turnID: string, e: CEEvent) => void;
  onStreamEnd: (turnID: string) => void;
}

function ceDataHandlerWith(turnID: string, callbacks: CEStreamCallbacks) {
  return (data: string) => {
    console.log("Received CE event", { turnID, data });
    try {
      const event: CEEvent = JSON.parse(data);
      if (!event || !event.turnid) {
        console.error("Invalid CE event, missing event.turnID", data);
        return;
      }
      if (event.turnid !== turnID) {
        console.error(
          `CE event turnID mismatch, expected ${turnID} but got ${event.turnid}`,
          data,
        );
        return;
      }
      switch (event.object) {
        case "llm.try.run.start":
          callbacks.onLLMTryRunStart(turnID, event);
          break;

        case "llm.try.run.end":
          callbacks.onLLMTryRunEnd(turnID, event);
          break;

        case "llm.try.run.failed":
          callbacks.onLLMTryRunFailed(turnID, event);
          break;

        case "research.turn.started":
          callbacks.onResearchTurnStarted(turnID, event);
          break;

        case "research.step.update":
          callbacks.onResearchStepUpdate(turnID, event);
          break;

        case "research.reasoning.delta":
          callbacks.onResearchReasoningDelta(turnID, event);
          break;

        case "research.source.added":
          callbacks.onResearchSourceAdded(turnID, event);
          break;

        case "resource.material":
          callbacks.onResourceMaterial(turnID, event);
          break;

        case "tool.call.request":
          callbacks.onToolCallRequest(turnID, event);
          break;

        case "tool.call.response":
          callbacks.onToolCallResponse(turnID, event);
          break;

        default:
          console.error(`CE event not supported: ${JSON.stringify(event)}`);
          break;
      }
    } catch (e) {
      console.error("Failed to parse CE event", e, data);
    }
  };
}

function openaiDataHandlerWith(turnID: string, callbacks: CEStreamCallbacks) {
  return (data: string) => {
    console.log("Received OpenAI data chunk", { turnID, data });
    if (data === "[DONE]") {
      return;
    }

    try {
      const chunk: OpenAIChunk = JSON.parse(data);
      callbacks.onOpenaiChunk(turnID, chunk);
    } catch (e) {
      console.error("Failed to parse data chunk", e, data);
    }
  };
}

function endHandlerWith(turnID: string, callbacks: CEStreamCallbacks) {
  return () => {
    callbacks.onStreamEnd(turnID);
  };
}

export function processCEStream(
  turnID: string,
  reader: ReadableStreamDefaultReader<Uint8Array>,
  callbacks: CEStreamCallbacks,
): Promise<void> {
  const emitter = createStreamEmitter(reader, {
    prefixes: {
      ce: "ce: ", // ce event prefix
      data: "data: ", // openai event prefix
    },
  });
  emitter
    .on("ce", ceDataHandlerWith(turnID, callbacks))
    .on("data", openaiDataHandlerWith(turnID, callbacks))
    .on("end", endHandlerWith(turnID, callbacks));
  return emitter.process();
}

export function processCEText(
  turnID: string,
  text: string,
  callbacks: CEStreamCallbacks,
): void {
  const lines = text.split("\n\n");
  for (const line of lines) {
    if (line.trim().length <= 0) {
      continue;
    }

    if (line.startsWith("data: ")) {
      openaiDataHandlerWith(turnID, callbacks)(line.slice(6));
    } else if (line.startsWith("ce: ")) {
      ceDataHandlerWith(turnID, callbacks)(line.slice(4));
    }
  }
}

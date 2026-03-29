/** biome-ignore-all lint/suspicious/noExplicitAny: stream process lib */
export type StreamEventType = string | "error" | "end";

export type EventHandler = (data: any) => void;
export type ErrorHandler = (error: Error) => void;
export type EndHandler = () => void;

export interface StreamEmitter {
  on(event: string, handler: EventHandler): StreamEmitter;
  on(event: "error", handler: ErrorHandler): StreamEmitter;
  on(event: "end", handler: EndHandler): StreamEmitter;
  off(event: string, handler: GenericHandler): StreamEmitter;
  off(event: "error", handler: GenericHandler): StreamEmitter;
  off(event: "end", handler: GenericHandler): StreamEmitter;
  once(event: string, handler: EventHandler): StreamEmitter;
  once(event: "error", handler: ErrorHandler): StreamEmitter;
  once(event: "end", handler: EndHandler): StreamEmitter;
  process(): Promise<void>;
  abort(): void;
}

export interface StreamConfig {
  prefixes: Record<string, string>;
}

type GenericHandler = EventHandler | ErrorHandler | EndHandler;

const DEFAULT_CONFIG: StreamConfig = {
  prefixes: {
    data: "data: ",
    ce: "ce: ",
  },
};

export function createStreamEmitter(
  reader: ReadableStreamDefaultReader<Uint8Array>,
  config: Partial<StreamConfig> = {},
): StreamEmitter {
  const mergedConfig: StreamConfig = {
    prefixes: { ...DEFAULT_CONFIG.prefixes, ...config.prefixes },
  };

  const handlers: Record<string, Set<GenericHandler>> = {};
  const onceHandlers: Record<string, Set<GenericHandler>> = {};

  let aborted = false;
  const decoder = new TextDecoder();
  let buffer = "";

  const emit = (event: string, data?: any, error?: Error) => {
    const eventHandlers = handlers[event];
    const eventOnceHandlers = onceHandlers[event];
    const allHandlers = [
      ...(eventHandlers ? [...eventHandlers] : []),
      ...(eventOnceHandlers ? [...eventOnceHandlers] : []),
    ];

    for (const handler of allHandlers) {
      try {
        if (event === "error" && error) {
          (handler as ErrorHandler)(error);
        } else if (event === "end") {
          (handler as EndHandler)();
        } else if (data !== undefined) {
          (handler as EventHandler)(data);
        }
      } catch (e) {
        console.error(`Error in ${event} handler:`, e);
      }
    }

    delete onceHandlers[event];
  };

  const on = (event: string, handler: GenericHandler): StreamEmitter => {
    if (!handlers[event]) {
      handlers[event] = new Set();
    }
    handlers[event]?.add(handler);
    return emitter;
  };

  const off = (event: string, handler: GenericHandler): StreamEmitter => {
    handlers[event]?.delete(handler);
    onceHandlers[event]?.delete(handler);
    return emitter;
  };

  const once = (event: string, handler: GenericHandler): StreamEmitter => {
    if (!onceHandlers[event]) {
      onceHandlers[event] = new Set();
    }
    onceHandlers[event]?.add(handler);
    return emitter;
  };

  const process = async (): Promise<void> => {
    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n\n");
        buffer = lines.pop() || "";

        for (const line of lines) {
          if (aborted) return;

          if (!line.trim()) continue;

          // NOTE: below we will only apply the first prefix matched
          for (const [event, prefix] of Object.entries(mergedConfig.prefixes)) {
            if (line.startsWith(prefix)) {
              const data = line.slice(prefix.length);
              emit(event, data);
              break;
            }
          }
        }
      }
    } catch (error) {
      emit(
        "error",
        undefined,
        error instanceof Error ? error : new Error(String(error)),
      );
    } finally {
      emit("end");
    }
  };

  const abort = (): void => {
    aborted = true;
    reader.cancel();
  };

  const emitter: StreamEmitter = {
    on,
    off,
    once,
    process,
    abort,
  };

  return emitter;
}

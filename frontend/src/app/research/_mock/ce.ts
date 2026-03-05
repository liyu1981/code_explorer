import { MOCK_REPORT_1, MOCK_SOURCES_1 } from "./data";

export interface OpenAIChunk {
  id: string;
  object: "chat.completion.chunk";
  choices: {
    delta: {
      content?: string;
    };
    finish_reason: string | null;
  }[];
}

export interface CEEvent {
  object: string;
  id?: string;
  status?: "pending" | "active" | "completed";
  content?: string;
  tool?: string;
  params?: any;
  response?: any;
  source?: any;
}

// Helper to chunk text for simulation
export function chunkText(text: string, size = 10): string[] {
  const chunks: string[] = [];
  for (let i = 0; i < text.length; i += size) {
    chunks.push(text.slice(i, i + size));
  }
  return chunks;
}

export const getMockStream = (query: string) => {
  const stream: string[] = [];

  // 1. Initial Step Updates
  stream.push(
    `ce: ${JSON.stringify({ object: "research.step.update", id: "1", status: "active" })}`,
  );
  stream.push(
    `ce: ${JSON.stringify({ object: "research.reasoning.delta", content: `Initializing deep research for: ${query}\n` })}`,
  );

  // 2. Tool call simulation
  stream.push(
    `ce: ${JSON.stringify({ object: "tool.call.request", tool: "grep_search", params: { pattern: "useSocket" } })}`,
  );
  stream.push(
    `ce: ${JSON.stringify({ object: "research.reasoning.delta", content: "Searching for WebSocket patterns...\n" })}`,
  );
  stream.push(
    `ce: ${JSON.stringify({ object: "tool.call.response", tool: "grep_search", response: { matches: ["src/hooks/useSocket.ts"] } })}`,
  );

  // 3. Step transition
  stream.push(
    `ce: ${JSON.stringify({ object: "research.step.update", id: "1", status: "completed" })}`,
  );
  stream.push(
    `ce: ${JSON.stringify({ object: "research.step.update", id: "2", status: "active" })}`,
  );

  // 4. Source added
  for (const source of MOCK_SOURCES_1) {
    stream.push(
      `ce: ${JSON.stringify({ object: "research.source.added", source })}`,
    );
  }

  // 5. Report streaming (data: prefix)
  const reportChunks = chunkText(MOCK_REPORT_1, 20);
  for (const chunk of reportChunks) {
    stream.push(
      `data: ${JSON.stringify({
        id: "gen-123",
        object: "chat.completion.chunk",
        choices: [{ delta: { content: chunk }, finish_reason: null }],
      })}`,
    );
  }

  // 6. Finalize
  stream.push(
    `ce: ${JSON.stringify({ object: "research.step.update", id: "2", status: "completed" })}`,
  );
  stream.push(
    `ce: ${JSON.stringify({ object: "research.step.update", id: "3", status: "completed" })}`,
  );
  stream.push(
    `data: ${JSON.stringify({ choices: [{ delta: {}, finish_reason: "stop" }] })}`,
  );

  return stream;
};

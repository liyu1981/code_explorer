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
  label?: string;
  content?: string;
  tool?: string;
  params?: any;
  response?: any;
  source?: any;
  resource?: any;
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
    `ce: ${JSON.stringify({
      object: "research.step.update",
      id: "thinking",
      label: "Thinking about the research plan",
      status: "active",
    })}`,
  );
  stream.push(
    `ce: ${JSON.stringify({
      object: "research.step.update",
      id: "thinking",
      label: "Thinking about the research plan",
      status: "completed",
    })}`,
  );
  stream.push(
    `ce: ${JSON.stringify({
      object: "research.step.update",
      id: "1",
      label: "Searching codebase for context",
      status: "active",
    })}`,
  );
  stream.push(
    `ce: ${JSON.stringify({
      object: "research.reasoning.delta",
      content: `Initializing deep research for: ${query}\n
[ce] 2026/03/13 14:30:28 GET /api/research/sessions 695.932s
[ce] 2026/03/13 14:30:28 GET /api/research/sessions/C_tovrAmB3/reports 223.367s
[ce] 2026/03/13 14:30:28 GET /api/research/sessions 473.037s
[ce] 2026/03/13 14:30:28 GET /api/research/sessions/C_tovrAmB3/reports 194.403s
[ce] 2026/03/13 14:30:28 GET /api/research/sessions 264.856s
[ce] 2026/03/13 14:30:28 GET /api/research/sessions/C_tovrAmB3/reports 188.806s
[ce] 2026/03/13 14:30:28 GET /api/research/sessions 296.193s
[ce] 2026/03/13 14:30:28 GET /api/research/sessions/C_tovrAmB3/reports 251.606s
[ce] 2026/03/13 14:30:28 GET /api/research/sessions 207.222s
[ce] 2026/03/13 14:30:28 GET /api/research/sessions/C_tovrAmB3/reports 94.421s\n`,
    })}`,
  );

  // 2. Tool call simulation
  stream.push(
    `ce: ${JSON.stringify({
      object: "tool.call.request",
      tool: "grep_search",
      params: { pattern: "useSocket" },
    })}`,
  );
  stream.push(
    `ce: ${JSON.stringify({
      object: "research.reasoning.delta",
      content: "Searching for WebSocket patterns...\n",
    })}`,
  );
  stream.push(
    `ce: ${JSON.stringify({
      object: "tool.call.response",
      tool: "grep_search",
      response: { matches: ["src/hooks/useSocket.ts"] },
    })}`,
  );

  // 3. Step transition
  stream.push(
    `ce: ${JSON.stringify({
      object: "research.step.update",
      id: "1",
      label: "Searching codebase for context",
      status: "completed",
    })}`,
  );
  stream.push(
    `ce: ${JSON.stringify({
      object: "research.step.update",
      id: "2",
      label: "Analyzing retrieved code chunks",
      status: "active",
    })}`,
  );

  // 4. Source added (using resource.material now)
  for (const source of MOCK_SOURCES_1) {
    stream.push(
      `ce: ${JSON.stringify({
        object: "resource.material",
        resource: {
          ...source,
          start_line: 1,
          end_line: 10,
        },
      })}`,
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
    `ce: ${JSON.stringify({
      object: "research.step.update",
      id: "2",
      label: "Analyzing retrieved code chunks",
      status: "completed",
    })}`,
  );
  stream.push(
    `ce: ${JSON.stringify({
      object: "research.step.update",
      id: "3",
      label: "Synthesizing deep research report",
      status: "completed",
    })}`,
  );
  stream.push(
    `data: ${JSON.stringify({ choices: [{ delta: {}, finish_reason: "stop" }] })}`,
  );

  return stream;
};

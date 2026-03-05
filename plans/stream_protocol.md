# Research Stream Data Protocol (Dual-Prefix)

The Research UI utilizes a streaming protocol that separates standard OpenAI-compatible text deltas from custom research-specific events using distinct line prefixes. This allows for clean structural separation and simplified parsing.

## Transport
- **Protocol**: Extended Server-Sent Events (SSE) / Raw Chunked HTTP
- **Content-Type**: `text/event-stream`
- **Prefixes**:
  - `data: <JSON>`: Standard OpenAI Chat Completion chunks.
  - `ce: <JSON>`: Custom Code Explorer (CE) research and tool-call events.

---

## 1. OpenAI Chunks (`data:`)
Used for streaming the primary markdown research report.

**Schema**: Standard OpenAI `chat.completion.chunk`
```json
data: {"id":"...","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"## Result\n"},"finish_reason":null}]}
```

---

## 2. Code Explorer Chunks (`ce:`)
Used for all custom research-specific metadata, status updates, and tool interactions.

### Reasoning Trace (`object: research.reasoning.delta`)
Granular thought process deltas.
```json
ce: {"object": "research.reasoning.delta", "content": "Analyzing file system..."}
```

### Step Updates (`object: research.step.update`)
Updates high-level UI progress indicators.
```json
ce: {"object": "research.step.update", "id": "step-1", "status": "completed"}
```

### Tool Interaction (`object: tool.call.request` | `object: tool.call.response`)
Full transparency of agent tool usage.
```json
// Request
ce: {"object": "tool.call.request", "tool": "grep_search", "params": {"pattern": "func"}}

// Response
ce: {"object": "tool.call.response", "tool": "grep_search", "response": {"matches": 5}}
```

### Source Material Added (`object: research.source.added`)
Pushes identified code snippets to the UI.
```json
ce: {
  "object": "research.source.added", 
  "source": {
    "id": "src-1",
    "path": "pkg/server/web.go",
    "snippet": "..."
  }
}
```

---

## 3. Implementation Logic

### Backend (Go)
The backend will have two distinct Marshalling paths:
- `SendOpenAIChunk(content string)`: Wraps content in the `data:` prefix.
- `SendCEEvent(event interface{})`: Serializes custom structs into the `ce:` prefix.

### Frontend (TypeScript)
The stream reader will switch logic based on the line prefix:
```typescript
const line = await readLine();
if (line.startsWith('data: ')) {
  const chunk = JSON.parse(line.slice(6));
  updateReport(chunk.choices[0].delta.content);
} else if (line.startsWith('ce: ')) {
  const event = JSON.parse(line.slice(4));
  handleCEEvent(event);
}
```

## 4. Completion
The stream closes with a standard `data: [DONE]` or by closing the HTTP connection after a final `ce:` event.

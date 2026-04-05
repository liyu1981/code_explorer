import test, { describe } from "node:test";
import assert from "node:assert";

import { processCEStream } from "./cestream";

const MOCK_API = "http://localhost:12345/api/mock/research";

describe("/api/mock/research e2e", () => {
  test("processCEStream processes mock stream from backend", async () => {
    const events: { type: string; content: string }[] = [];

    const turnID = "test-stream-turn";

    const response = await fetch(MOCK_API, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query: "How does the research endpoint work?",
        sessionId: "",
        turnId: turnID,
      }),
    });

    assert.ok(response.ok, `Expected ok response, got ${response.status}`);
    assert.ok(response.body, "Response body is null");

    const reader = response.body.getReader();

    await processCEStream(turnID, reader, {
      onOpenaiChunk: (_, data) => {
        const content = data.choices?.[0]?.delta?.content ?? "";
        console.log("[OpenAI Chunk]", content);
        events.push({ type: "openai_chunk", content });
      },
      onLLMTryRunStart: (_, e) => {
        console.log("[LLM Try Run Start]", e.object, e.tryid);
        events.push({ type: "llm_try_run_start", content: e.object });
      },
      onLLMTryRunEnd: (_, e) => {
        console.log("[LLM Try Run End]", e.object, e.tryid);
        events.push({ type: "llm_try_run_end", content: e.object });
      },
      onLLMTryRunFailed: (_, e) => {
        console.log("[LLM Try Run Failed]", e.object, e.tryid);
        events.push({ type: "llm_try_run_failed", content: e.object });
      },
      onResearchTurnStarted: (_, e) => {
        console.log("[Turn Started]", e.query);
        events.push({ type: "turn_started", content: e.query ?? "" });
      },
      onResearchStepUpdate: (_, e) => {
        console.log("[Step Update]", e.label, e.status);
        events.push({
          type: "step_update",
          content: `${e.label} -> ${e.status}`,
        });
      },
      onResearchReasoningDelta: (_, e) => {
        console.log("[Reasoning]", e.content);
        events.push({ type: "reasoning", content: e.content ?? "" });
      },
      onResearchSourceAdded: (_, e) => {
        console.log("[Source Added]", e.source?.path, e.source?.snippet);
        events.push({
          type: "source_added",
          content: e.source?.path ?? "",
        });
      },
      onResourceMaterial: (_, e) => {
        console.log("[Resource Material]", e.resource?.path);
        events.push({
          type: "resource_material",
          content: e.resource?.path ?? "",
        });
      },
      onStreamEnd: (_) => {
        console.log("[Stream End]");
        events.push({ type: "stream_end", content: "" });
      },
    });

    // Verify events were received
    const types = events.map((e) => e.type);
    console.log("\nAll events received:", types);

    assert.ok(types.includes("turn_started"), "Should have turn_started");
    assert.ok(types.includes("step_update"), "Should have step_update events");
    assert.ok(types.includes("reasoning"), "Should have reasoning events");
    assert.ok(types.includes("openai_chunk"), "Should have openai_chunk event");

    const stepEvents = events.filter((e) => e.type === "step_update");
    const completedSteps = stepEvents.filter((e) =>
      e.content.includes("completed"),
    );
    assert.ok(completedSteps.length > 0, "Should have completed steps");
  });
});

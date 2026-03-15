package task

import (
	"context"

	"github.com/liyu1981/code_explorer/pkg/agent"
	"github.com/liyu1981/code_explorer/pkg/codemogger"
	"github.com/liyu1981/code_explorer/pkg/db"
)

func RegisterQueueHandlers(
	m *Manager,
	index *codemogger.CodeIndex,
	agentFactory *agent.AgentFactory,
	publishFn func(topic string, payload any),
) {
	m.RegisterHandler("codemogger-index", func(ctx context.Context, task *db.Task, updateProgress func(progress int, message string)) error {
		return index.HandleIndexTask(ctx, task, updateProgress)
	})

	m.RegisterHandler("knowledge-wiki-plan", func(ctx context.Context, taskItem *db.Task, updateProgress func(progress int, message string)) error {
		return HandleKnowledgeWikiPlanTask(ctx, index, taskItem, m, agentFactory, updateProgress)
	})

	m.RegisterHandler("knownledge-wiki-build", func(ctx context.Context, taskItem *db.Task, updateProgress func(progress int, message string)) error {
		return HandleKnowledgeWikiBuildTask(ctx, index, taskItem, agentFactory, updateProgress)
	})

	m.RegisterHandler("summarize-topic", func(ctx context.Context, taskItem *db.Task, updateProgress func(progress int, message string)) error {
		return HandleSummarizeTopicTask(ctx, index, taskItem, agentFactory, updateProgress, func(sessionId string, title string) {
			publishFn("research", map[string]any{
				"type":      "research.session.updated",
				"sessionId": sessionId,
				"title":     title,
			})
		})
	})
}

package util

import "context"

type contextKey string

const (
	InitiatorIDKey contextKey = "initiator_id"
)

func WithInitiatorID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, InitiatorIDKey, id)
}

func GetInitiatorID(ctx context.Context) string {
	if id, ok := ctx.Value(InitiatorIDKey).(string); ok {
		return id
	}
	return ""
}

package requestlogger

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
)

const ID graft.ID = "request_logger"

// contextKey for request-specific data
type contextKey struct{}

var requestIDKey = contextKey{}

type Output struct {
	RequestID string
	StartTime time.Time
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{}, // No dependencies - request-scoped
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	reqID := ctx.Value(requestIDKey).(string)
	fmt.Printf("[request_logger] Logging request %s\n", reqID)

	return Output{
		RequestID: reqID,
		StartTime: time.Now(),
	}, nil
}

// SetRequestID adds request ID to context
func SetRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

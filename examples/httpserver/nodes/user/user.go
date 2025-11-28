package user

import (
	"context"
	"fmt"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/httpserver/nodes/db"
	"github.com/grindlemire/graft/examples/httpserver/nodes/requestlogger"
)

const ID graft.ID = "user"

// contextKey for user ID
type contextKey struct{}

var userIDKey = contextKey{}

type Output struct {
	UserID   string
	Username string
	Email    string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{db.ID, requestlogger.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	userID := ctx.Value(userIDKey).(string)

	dbConn, err := graft.Dep[db.Output](ctx, db.ID)
	if err != nil {
		return Output{}, err
	}

	reqLog, err := graft.Dep[requestlogger.Output](ctx, requestlogger.ID)
	if err != nil {
		return Output{}, err
	}

	fmt.Printf("[user_handler] Fetching user %s (request: %s, db execution: #%d)\n",
		userID, reqLog.RequestID, dbConn.ExecutionNum)

	// Simulate DB query
	return Output{
		UserID:   userID,
		Username: "user_" + userID,
		Email:    userID + "@example.com",
	}, nil
}

// SetUserID adds user ID to context
func SetUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

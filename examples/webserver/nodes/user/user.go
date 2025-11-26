package user

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/webserver/nodes/cache"
	"github.com/grindlemire/graft/examples/webserver/nodes/db"
)

const ID = "user"

type Output struct {
	UserID   string
	Username string
	Email    string
}

// SetUserID allows setting the user ID for the request context
type RequestContext struct {
	UserID string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []string{db.ID, cache.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	dbOut, err := graft.Dep[db.Output](ctx, db.ID)
	if err != nil {
		return Output{}, err
	}

	cacheOut, err := graft.Dep[cache.Output](ctx, cache.ID)
	if err != nil {
		return Output{}, err
	}

	// Get user ID from context if set
	userID, err := GetUserID(ctx)
	if err != nil {
		return Output{}, err
	}

	fmt.Printf("[user] Fetching user %s (db=%v, cache=%v)...\n",
		userID, dbOut.Connected, cacheOut.Connected)
	time.Sleep(25 * time.Millisecond)

	fmt.Println("[user] Done")
	return Output{
		UserID:   userID,
		Username: "john_doe",
		Email:    "john@example.com",
	}, nil
}

type requestContextKey struct{}

// GetUserID retrieves the user ID from the context if set
func GetUserID(ctx context.Context) (string, error) {
	if reqCtx, ok := ctx.Value(requestContextKey{}).(RequestContext); ok {
		return reqCtx.UserID, nil
	}
	return "", fmt.Errorf("user ID not found in context")
}

// SetUserID sets the user ID in the context
func SetUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, requestContextKey{}, RequestContext{UserID: userID})
}

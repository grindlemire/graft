package secrets

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/complex/nodes/env"
)

const ID graft.ID = "secrets"

type Output struct {
	JWTSecret   string
	APIKey      string
	DatabasePwd string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{env.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	envOut, err := graft.Dep[env.Output](ctx, env.ID)
	if err != nil {
		return Output{}, err
	}

	fmt.Printf("[secrets] Loading secrets for %s...\n", envOut.Environment)
	time.Sleep(100 * time.Millisecond)

	fmt.Println("[secrets] Done")
	return Output{
		JWTSecret:   "super-secret-jwt-key",
		APIKey:      "api-key-12345",
		DatabasePwd: "db-password",
	}, nil
}

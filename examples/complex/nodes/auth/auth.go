package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/complex/nodes/logger"
	"github.com/grindlemire/graft/examples/complex/nodes/secrets"
)

const ID graft.ID = "auth"

type Output struct {
	Ready        bool
	TokenExpiry  time.Duration
	JWTSecretLen int
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{secrets.ID, logger.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	sec, err := graft.Dep[secrets.Output](ctx, secrets.ID)
	if err != nil {
		return Output{}, err
	}

	log, err := graft.Dep[logger.Output](ctx, logger.ID)
	if err != nil {
		return Output{}, err
	}

	fmt.Printf("[auth] Initializing auth (log level: %s)...\n", log.Level)
	time.Sleep(90 * time.Millisecond)

	fmt.Println("[auth] Done")
	return Output{
		Ready:        true,
		TokenExpiry:  24 * time.Hour,
		JWTSecretLen: len(sec.JWTSecret),
	}, nil
}

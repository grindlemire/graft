package handlers

import (
	"context"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/testcases/importpaths/internal/services/auth/providers"
)

const ID graft.ID = "handler"

type Output struct{}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{providers.OAuthProviderID},
		Run: func(ctx context.Context) (Output, error) {
			// Using a deeply nested import path
			_, _ = graft.Dep[providers.OAuthToken](ctx)
			return Output{}, nil
		},
	})
}


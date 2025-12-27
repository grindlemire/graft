package providers

import (
	"context"

	"github.com/grindlemire/graft"
)

const OAuthProviderID graft.ID = "oauth-provider"

type OAuthToken struct {
	Token string
}

func init() {
	graft.Register(graft.Node[OAuthToken]{
		ID: OAuthProviderID,
		Run: func(ctx context.Context) (OAuthToken, error) {
			return OAuthToken{Token: "abc123"}, nil
		},
	})
}


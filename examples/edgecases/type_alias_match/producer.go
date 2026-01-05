package type_alias_match

import (
	"context"

	"github.com/grindlemire/graft"
)

// Producer outputs a UserID (which is an alias for string)
func init() {
	graft.Register(graft.Node[UserID]{
		ID: "uid",
		Run: func(ctx context.Context) (UserID, error) {
			return "user-123", nil
		},
	})
}

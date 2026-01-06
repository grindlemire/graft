package same_package_deps

import (
	"context"

	"github.com/grindlemire/graft"
)

// All types and nodes in the same package

type Config struct {
	Port int
}

type Database struct {
	ConnString string
}

type App struct {
	DB *Database
}

// Config node - no dependencies
func init() {
	graft.Register(graft.Node[Config]{
		ID: "config",
		Run: func(ctx context.Context) (Config, error) {
			return Config{Port: 8080}, nil
		},
	})
}

// Database node - depends on Config
func init() {
	graft.Register(graft.Node[Database]{
		ID:        "db",
		DependsOn: []graft.ID{"config"},
		Run: func(ctx context.Context) (Database, error) {
			cfg, _ := graft.Dep[Config](ctx)
			return Database{ConnString: "localhost:" + string(rune(cfg.Port))}, nil
		},
	})
}

// App node - depends on Database
func init() {
	graft.Register(graft.Node[App]{
		ID:        "app",
		DependsOn: []graft.ID{"db"},
		Run: func(ctx context.Context) (App, error) {
			db, _ := graft.Dep[Database](ctx)
			return App{DB: &db}, nil
		},
	})
}

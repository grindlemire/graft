package admin

import (
	"context"
	"fmt"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/httpserver/nodes/config"
	"github.com/grindlemire/graft/examples/httpserver/nodes/requestlogger"
)

const ID graft.ID = "admin"

type Output struct {
	ServerUptime  string
	ConfigExecNum int32
	RequestID     string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []graft.ID{config.ID, requestlogger.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	cfg, err := graft.Dep[config.Output](ctx)
	if err != nil {
		return Output{}, err
	}

	reqLog, err := graft.Dep[requestlogger.Output](ctx)
	if err != nil {
		return Output{}, err
	}

	fmt.Printf("[admin_handler] Admin request %s (config execution: #%d)\n",
		reqLog.RequestID, cfg.ExecutionNum)

	return Output{
		ServerUptime:  fmt.Sprintf("Server started at %s", cfg.StartupTime),
		ConfigExecNum: cfg.ExecutionNum,
		RequestID:     reqLog.RequestID,
	}, nil
}

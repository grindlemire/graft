package aggregator

import (
	"context"
	"fmt"
	"time"

	"github.com/grindlemire/graft"
	"github.com/grindlemire/graft/examples/fanout/nodes/svc1"
	"github.com/grindlemire/graft/examples/fanout/nodes/svc2"
	"github.com/grindlemire/graft/examples/fanout/nodes/svc3"
	"github.com/grindlemire/graft/examples/fanout/nodes/svc4"
	"github.com/grindlemire/graft/examples/fanout/nodes/svc5"
)

const ID = "aggregator"

type Output struct {
	TotalServices int
	TotalResult   int
	Services      []string
}

func init() {
	graft.Register(graft.Node[Output]{
		ID:        ID,
		DependsOn: []string{svc1.ID, svc2.ID, svc3.ID, svc4.ID, svc5.ID},
		Run:       run,
	})
}

func run(ctx context.Context) (Output, error) {
	s1, err := graft.Dep[svc1.Output](ctx, svc1.ID)
	if err != nil {
		return Output{}, err
	}
	s2, err := graft.Dep[svc2.Output](ctx, svc2.ID)
	if err != nil {
		return Output{}, err
	}
	s3, err := graft.Dep[svc3.Output](ctx, svc3.ID)
	if err != nil {
		return Output{}, err
	}
	s4, err := graft.Dep[svc4.Output](ctx, svc4.ID)
	if err != nil {
		return Output{}, err
	}
	s5, err := graft.Dep[svc5.Output](ctx, svc5.ID)
	if err != nil {
		return Output{}, err
	}

	services := []svc1.Output{
		{Name: s1.Name, Result: s1.Result},
		{Name: s2.Name, Result: s2.Result},
		{Name: s3.Name, Result: s3.Result},
		{Name: s4.Name, Result: s4.Result},
		{Name: s5.Name, Result: s5.Result},
	}

	fmt.Printf("[aggregator] Aggregating %d service results...\n", len(services))
	time.Sleep(50 * time.Millisecond)

	total := 0
	names := make([]string, len(services))
	for i, s := range services {
		total += s.Result
		names[i] = s.Name
	}

	fmt.Println("[aggregator] Done")
	return Output{
		TotalServices: len(services),
		TotalResult:   total,
		Services:      names,
	}, nil
}

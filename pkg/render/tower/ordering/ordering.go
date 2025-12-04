package ordering

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/dag"
)

type Orderer interface {
	OrderRows(g *dag.DAG) map[int][]string
}

type ContextOrderer interface {
	Orderer
	OrderRowsContext(ctx context.Context, g *dag.DAG) map[int][]string
}

type Quality int

const (
	QualityFast Quality = iota
	QualityBalanced
	QualityOptimal
)

const (
	DefaultTimeoutFast     = 100 * time.Millisecond
	DefaultTimeoutBalanced = 5 * time.Second
	DefaultTimeoutOptimal  = 60 * time.Second
)

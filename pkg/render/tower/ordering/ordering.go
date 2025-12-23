package ordering

import (
	"context"
	"time"

	"github.com/matzehuels/stacktower/pkg/dag"
)

// Orderer is an interface for horizontal row ordering algorithms.
// An orderer determines the horizontal sequence of nodes in each row
// to minimize edge crossings.
type Orderer interface {
	OrderRows(g *dag.DAG) map[int][]string
}

// ContextOrderer is an Orderer that supports cancellation and timeouts
// via a context.
type ContextOrderer interface {
	Orderer
	OrderRowsContext(ctx context.Context, g *dag.DAG) map[int][]string
}

// Quality represents the desired trade-off between ordering speed and quality.
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

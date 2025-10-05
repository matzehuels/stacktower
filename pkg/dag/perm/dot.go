package perm

import (
	"bytes"
	"context"
	"fmt"

	"github.com/goccy/go-graphviz"
)

func (t *PQTree) ToDOT(labels []string) string {
	var buf bytes.Buffer
	buf.WriteString("digraph PQTree {\n")
	buf.WriteString("  rankdir=TB;\n")
	buf.WriteString("  bgcolor=\"transparent\";\n")
	buf.WriteString("  node [fontname=\"SF Mono, Menlo, monospace\", fontsize=14, style=filled, fillcolor=white];\n")
	buf.WriteString("  edge [arrowhead=none];\n\n")

	if t.root != nil {
		t.writeDOTNode(&buf, t.root, 0, labels)
	}

	buf.WriteString("}\n")
	return buf.String()
}

func (t *PQTree) writeDOTNode(buf *bytes.Buffer, n *pqNode, id int, labels []string) int {
	nodeID := fmt.Sprintf("n%d", id)
	next := id + 1

	switch n.kind {
	case leafNode:
		label := t.nodeString(n, labels)
		fmt.Fprintf(buf, "  %s [label=%q, shape=box, style=\"filled,rounded\"];\n", nodeID, label)

	case pNode:
		fmt.Fprintf(buf, "  %s [label=\"P\", shape=ellipse];\n", nodeID)
		for _, c := range n.children {
			fmt.Fprintf(buf, "  %s -> n%d;\n", nodeID, next)
			next = t.writeDOTNode(buf, c, next, labels)
		}

	case qNode:
		fmt.Fprintf(buf, "  %s [label=\"Q\", shape=box];\n", nodeID)
		for _, c := range n.children {
			fmt.Fprintf(buf, "  %s -> n%d;\n", nodeID, next)
			next = t.writeDOTNode(buf, c, next, labels)
		}
	}

	return next
}

func (t *PQTree) RenderSVG(labels []string) ([]byte, error) {
	dot := t.ToDOT(labels)

	gv, err := graphviz.New(context.Background())
	if err != nil {
		return nil, fmt.Errorf("init graphviz: %w", err)
	}
	defer gv.Close()

	g, err := graphviz.ParseBytes([]byte(dot))
	if err != nil {
		return nil, fmt.Errorf("parse DOT: %w", err)
	}
	defer g.Close()

	var buf bytes.Buffer
	if err := gv.Render(context.Background(), g, graphviz.SVG, &buf); err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}
	return buf.Bytes(), nil
}

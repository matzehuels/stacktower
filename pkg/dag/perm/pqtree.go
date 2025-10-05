package perm

import "slices"

type PQTree struct {
	root   *pqNode
	leaves []*pqNode
}

type nodeKind int

const (
	leafNode nodeKind = iota
	pNode
	qNode
)

type markKind int

const (
	unmarked markKind = iota
	empty
	full
	partial
)

type pqNode struct {
	kind         nodeKind
	value        int
	children     []*pqNode
	parent       *pqNode
	mark         markKind
	fullCount    int
	partialCount int
}

func NewPQTree(n int) *PQTree {
	if n == 0 {
		return &PQTree{}
	}
	if n == 1 {
		leaf := &pqNode{kind: leafNode, value: 0}
		return &PQTree{root: leaf, leaves: []*pqNode{leaf}}
	}

	leaves := make([]*pqNode, n)
	for i := range leaves {
		leaves[i] = &pqNode{kind: leafNode, value: i}
	}

	root := &pqNode{kind: pNode, children: slices.Clone(leaves)}
	for _, child := range leaves {
		child.parent = root
	}

	return &PQTree{root: root, leaves: leaves}
}

func (t *PQTree) Reduce(constraint []int) bool {
	if t.root == nil || len(constraint) <= 1 || len(constraint) == len(t.leaves) {
		return true
	}

	t.clearMarks(t.root)
	for _, elem := range constraint {
		if elem >= 0 && elem < len(t.leaves) {
			t.leaves[elem].mark = full
		}
	}

	if t.bubbleUp(t.root) == empty {
		return true
	}

	return t.reduce(t.root)
}

func (t *PQTree) clearMarks(n *pqNode) {
	n.mark = unmarked
	n.fullCount = 0
	n.partialCount = 0
	for _, c := range n.children {
		t.clearMarks(c)
	}
}

func (t *PQTree) bubbleUp(n *pqNode) markKind {
	if n.kind == leafNode {
		if n.mark == unmarked {
			n.mark = empty
		}
		return n.mark
	}

	n.fullCount = 0
	n.partialCount = 0
	for _, c := range n.children {
		switch t.bubbleUp(c) {
		case full:
			n.fullCount++
		case partial:
			n.partialCount++
		}
	}

	switch {
	case n.fullCount == len(n.children):
		n.mark = full
	case n.fullCount == 0 && n.partialCount == 0:
		n.mark = empty
	default:
		n.mark = partial
	}
	return n.mark
}

func (t *PQTree) reduce(n *pqNode) bool {
	if n.mark == full || n.mark == empty {
		return true
	}

	switch n.kind {
	case leafNode:
		return true
	case pNode:
		return t.reducePNode(n)
	case qNode:
		return t.reduceQNode(n)
	}
	return false
}

func (t *PQTree) reducePartialChildren(n *pqNode) bool {
	for _, c := range n.children {
		if c.mark == partial && !t.reduce(c) {
			return false
		}
	}
	n.fullCount = 0
	n.partialCount = 0
	for _, c := range n.children {
		switch c.mark {
		case full:
			n.fullCount++
		case partial:
			n.partialCount++
		}
	}
	return true
}

func (t *PQTree) reducePNode(n *pqNode) bool {
	if !t.reducePartialChildren(n) {
		return false
	}

	var fullCh, emptyCh, partialCh []*pqNode
	for _, child := range n.children {
		switch child.mark {
		case full:
			fullCh = append(fullCh, child)
		case empty:
			emptyCh = append(emptyCh, child)
		case partial:
			partialCh = append(partialCh, child)
		}
	}

	if len(partialCh) > 1 {
		return false
	}

	if len(fullCh) == 0 {
		return true
	}

	if len(partialCh) == 0 {
		if len(fullCh) > 1 && len(emptyCh) > 0 {
			t.groupChildren(n, fullCh, pNode)
		}
		return true
	}

	return t.extendPartialChild(n, partialCh[0], fullCh)
}

func (t *PQTree) reduceQNode(n *pqNode) bool {
	if !t.reducePartialChildren(n) {
		return false
	}

	first, last := -1, -1
	var partialIdx []int
	for i, child := range n.children {
		switch child.mark {
		case full:
			if first < 0 {
				first = i
			}
			last = i
		case partial:
			partialIdx = append(partialIdx, i)
		}
	}

	if first < 0 {
		return true
	}

	for i := first; i <= last; i++ {
		if n.children[i].mark == empty {
			return false
		}
	}

	for _, idx := range partialIdx {
		if idx != first-1 && idx != last+1 {
			return false
		}
	}

	for _, idx := range partialIdx {
		if n.children[idx].kind == qNode {
			t.mergeQNodes(n, idx)
		}
	}
	return true
}

func (t *PQTree) groupChildren(parent *pqNode, group []*pqNode, kind nodeKind) {
	if len(group) <= 1 {
		return
	}

	node := &pqNode{
		kind:     kind,
		children: slices.Clone(group),
		parent:   parent,
		mark:     group[0].mark,
	}

	for _, child := range group {
		child.parent = node
	}

	groupSet := make(map[*pqNode]bool, len(group))
	for _, child := range group {
		groupSet[child] = true
	}

	newChildren := make([]*pqNode, 0, len(parent.children)-len(group)+1)
	inserted := false
	for _, child := range parent.children {
		if groupSet[child] {
			if !inserted {
				newChildren = append(newChildren, node)
				inserted = true
			}
		} else {
			newChildren = append(newChildren, child)
		}
	}
	parent.children = newChildren
}

func (t *PQTree) extendPartialChild(parent, partialChild *pqNode, fullSiblings []*pqNode) bool {
	if len(fullSiblings) == 0 {
		return true
	}

	var fullInPartial, emptyInPartial []*pqNode
	for _, child := range partialChild.children {
		if child.mark == full {
			fullInPartial = append(fullInPartial, child)
		} else {
			emptyInPartial = append(emptyInPartial, child)
		}
	}

	children := make([]*pqNode, 0, len(fullInPartial)+len(fullSiblings)+len(emptyInPartial))
	children = append(children, emptyInPartial...)
	children = append(children, fullInPartial...)
	children = append(children, fullSiblings...)

	qnode := &pqNode{
		kind:     qNode,
		children: children,
		parent:   parent,
		mark:     partial,
	}
	for _, child := range children {
		child.parent = qnode
	}

	toRemove := make(map[*pqNode]bool, len(fullSiblings)+1)
	toRemove[partialChild] = true
	for _, sibling := range fullSiblings {
		toRemove[sibling] = true
	}

	newChildren := make([]*pqNode, 0, len(parent.children)-len(toRemove)+1)
	replaced := false
	for _, child := range parent.children {
		if toRemove[child] {
			if !replaced {
				newChildren = append(newChildren, qnode)
				replaced = true
			}
		} else {
			newChildren = append(newChildren, child)
		}
	}
	parent.children = newChildren

	if len(parent.children) == 1 && parent == t.root {
		t.root = parent.children[0]
		t.root.parent = nil
	}

	return true
}

func (t *PQTree) mergeQNodes(parent *pqNode, idx int) {
	child := parent.children[idx]
	if child.kind != qNode {
		return
	}

	reverse := false
	if idx > 0 && parent.children[idx-1].mark == full {
		if len(child.children) > 0 && child.children[len(child.children)-1].mark == full {
			reverse = true
		}
	} else if idx < len(parent.children)-1 && parent.children[idx+1].mark == full {
		if len(child.children) > 0 && child.children[0].mark == full {
			reverse = true
		}
	}

	if reverse {
		slices.Reverse(child.children)
	}

	for _, grandchild := range child.children {
		grandchild.parent = parent
	}

	newChildren := make([]*pqNode, 0, len(parent.children)+len(child.children)-1)
	newChildren = append(newChildren, parent.children[:idx]...)
	newChildren = append(newChildren, child.children...)
	newChildren = append(newChildren, parent.children[idx+1:]...)
	parent.children = newChildren
}

func (t *PQTree) Enumerate(limit int) [][]int {
	if t.root == nil {
		return [][]int{{}}
	}

	var results [][]int
	t.enumerate(t.root, nil, limit, &results)
	return results
}

func (t *PQTree) enumerate(node *pqNode, prefix []int, limit int, results *[][]int) bool {
	if limit > 0 && len(*results) >= limit {
		return false
	}

	if node.kind == leafNode {
		*results = append(*results, append(slices.Clone(prefix), node.value))
		return true
	}

	perms := t.childPermutations(node)
	for _, perm := range perms {
		if limit > 0 && len(*results) >= limit {
			return false
		}
		if !t.enumerateChildren(perm, prefix, limit, results) {
			return false
		}
	}

	return true
}

func (t *PQTree) childPermutations(node *pqNode) [][]*pqNode {
	if node.kind == qNode {
		fwd := slices.Clone(node.children)
		rev := slices.Clone(node.children)
		slices.Reverse(rev)
		return [][]*pqNode{fwd, rev}
	}

	indexPerms := Generate(len(node.children), -1)
	result := make([][]*pqNode, len(indexPerms))
	for i, perm := range indexPerms {
		ordered := make([]*pqNode, len(perm))
		for j, idx := range perm {
			ordered[j] = node.children[idx]
		}
		result[i] = ordered
	}
	return result
}

func (t *PQTree) enumerateChildren(children []*pqNode, prefix []int, limit int, results *[][]int) bool {
	if len(children) == 0 {
		*results = append(*results, slices.Clone(prefix))
		return true
	}

	first := children[0]
	rest := children[1:]

	var firstPerms [][]int
	t.enumerate(first, nil, 0, &firstPerms)

	for _, fp := range firstPerms {
		if limit > 0 && len(*results) >= limit {
			return false
		}

		newPrefix := append(slices.Clone(prefix), fp...)
		if !t.enumerateChildren(rest, newPrefix, limit, results) {
			return false
		}
	}

	return true
}

func (t *PQTree) ValidCount() int {
	if t.root == nil {
		return 1
	}
	return t.countPerms(t.root)
}

func (t *PQTree) countPerms(node *pqNode) int {
	if node.kind == leafNode {
		return 1
	}

	product := 1
	for _, child := range node.children {
		product *= t.countPerms(child)
	}

	switch node.kind {
	case qNode:
		return 2 * product
	default:
		return Factorial(len(node.children)) * product
	}
}

func (t *PQTree) String() string {
	return t.StringWithLabels(nil)
}

func (t *PQTree) StringWithLabels(labels []string) string {
	if t.root == nil {
		return "(empty)"
	}
	return t.nodeString(t.root, labels)
}

func (t *PQTree) nodeString(n *pqNode, labels []string) string {
	if n.kind == leafNode {
		if n.value < len(labels) {
			return labels[n.value]
		}
		if n.value < 10 {
			return string(rune('0' + n.value))
		}
		return "(" + string(rune('a'+n.value-10)) + ")"
	}

	open, close := "{", "}"
	if n.kind == qNode {
		open, close = "[", "]"
	}

	s := open
	for i, child := range n.children {
		if i > 0 {
			s += " "
		}
		s += t.nodeString(child, labels)
	}
	return s + close
}

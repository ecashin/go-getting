// CLRS Chapter 13

package rbtree

type Tree struct {
	Parent, Left, Right *Tree
	Label               string
}

type Visitor func(t *Tree)

func (t *Tree) InOrder(visitor Visitor) {
	visitor(t)
}

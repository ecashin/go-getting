// CLRS Chapter 13

package rbtree

type Tree struct {
	Parent, Left, Right *Tree
	Label               string
}

type Visitor func(t *Tree, data interface{})

func (t *Tree) InOrder(visitor Visitor, data interface{}) {
	if t.Left != nil {
		t.Left.InOrder(visitor, data)
	}
	visitor(t, data)
	if t.Right != nil {
		t.Right.InOrder(visitor, data)
	}
}

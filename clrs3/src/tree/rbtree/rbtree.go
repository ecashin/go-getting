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

func (t *Tree) LeftRotate() *Tree {
	y := t.Right
	t.Right = y.Left
	if y.Left != nil {
		y.Left.Parent = t
	}
	y.Parent = t.Parent
	if t.Parent != nil {
		if t.Parent.Left == t {
			t.Parent.Left = y
		} else {
			t.Parent.Right = y
		}
	}
	y.Left = t
	t.Parent = y
	return y
}

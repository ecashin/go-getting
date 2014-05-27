package rbtree

import (
	"fmt"
	"testing"
)

func TestRbtree(test *testing.T) {
	t := &Tree{}
	t.Label = "x"
	t.Left = &Tree{t, nil, nil, "alpha"}
	t.Right = &Tree{t.Left, nil, nil, "y"}
	t.Right.Left = &Tree{t.Right, nil, nil, "beta"}
	t.Right.Right = &Tree{t.Right, nil, nil, "gamma"}
	t.InOrder(func(t *Tree) { fmt.Println(t.Label) })
}

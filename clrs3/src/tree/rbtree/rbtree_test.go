package rbtree

import (
	"bytes"
	"testing"
)

func makeTree() *Tree {
	t := &Tree{}
	t.Parent = nil
	t.Label = "x"
	t.Left = &Tree{t, nil, nil, "alpha"}
	t.Right = &Tree{t, nil, nil, "y"}
	t.Right.Left = &Tree{t.Right, nil, nil, "beta"}
	t.Right.Right = &Tree{t.Right, nil, nil, "gamma"}
	return t
}

const ref = "alpha,x,beta,y,gamma,"

func TestRbtree(test *testing.T) {
	var buf bytes.Buffer

	t := makeTree()
	t.InOrder(func(tt *Tree, d interface{}) {
		b := d.(*bytes.Buffer)
		b.WriteString(tt.Label + ",")
	}, &buf)
	res := buf.String()
	if res != ref {
		test.Errorf("expected \"%s\" but got \"%s\"", ref, res)
	}
}

func TestLeftRotate(test *testing.T) {
	var buf bytes.Buffer

	t := makeTree()
	t = t.LeftRotate()
	t.InOrder(func(tt *Tree, d interface{}) {
		b := d.(*bytes.Buffer)
		b.WriteString(tt.Label + ",")
	}, &buf)
	res := buf.String()
	if res != ref {
		test.Errorf("expected \"%s\" but got \"%s\"", ref, res)
	}
}

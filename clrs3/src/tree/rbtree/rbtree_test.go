package rbtree

import (
	"bytes"
	"testing"
)

func TestRbtree(test *testing.T) {
	var buf bytes.Buffer

	t := &Tree{}
	t.Label = "x"
	t.Left = &Tree{t, nil, nil, "alpha"}
	t.Right = &Tree{t.Left, nil, nil, "y"}
	t.Right.Left = &Tree{t.Right, nil, nil, "beta"}
	t.Right.Right = &Tree{t.Right, nil, nil, "gamma"}
	t.InOrder(func(tt *Tree, d interface{}) {
		b := d.(*bytes.Buffer)
		b.WriteString(tt.Label + ",")
	}, &buf)
	res := buf.String()
	const ref = "alpha,x,beta,y,gamma,"
	if res != ref {
		test.Errorf("expected \"%s\" but got \"%s\"", ref, res)
	}
}

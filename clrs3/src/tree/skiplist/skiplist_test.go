package skiplist

import (
	"testing"
)

func cmp(aa, bb interface{}) int {
	a := aa.(int)
	b := bb.(int)
	switch {
	case a > b:
		return -1
	case b > a:
		return 1
	default:
		return 0
	}
}

func TestSkipList(t *testing.T) {
	skp := SkipList{}
	skp.cmp = cmp
	skp.Insert(1)
}

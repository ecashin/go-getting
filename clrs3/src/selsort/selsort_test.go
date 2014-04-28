package selsort

import (
	"testing"
)

func TestSorting(t *testing.T) {
	a := []int64{2, 3, 4, 2, 1}
	ref := []int64{1, 2, 2, 3, 4}
	Sort(a)
	for i := 0; i < len(a); i++ {
		if a[i] != ref[i] {
			t.Errorf("a[%d] == %d", i, a[i])
		}
	}
}

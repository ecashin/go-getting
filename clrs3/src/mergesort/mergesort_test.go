package mergesort

import (
	"testing"
)

func TestMerge(t *testing.T) {
	a := []int64{1, 3, 5, 2, 4, 6}
	ref := []int64{1, 2, 3, 4, 5, 6}
	merge(a, 0, 3, 6)
	for i, v := range a {
		if v != ref[i] {
			t.Errorf("(a[%d]==%d) != (ref[%d]==%d)",
				i, a[i], i, ref[i])
		}
	}
}

func TestSorting(t *testing.T) {
	a := []int64{2, 3, 4, 2, 1}
	ref := []int64{1, 2, 2, 3, 4}
	Sort(a)
	for i, v := range a {
		if v != ref[i] {
			t.Errorf("(a[%d]==%d) != (ref[%d]==%d)",
				i, v, i, ref[i])
		}
	}
}

package bubblesort

import (
	"testing"
)

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

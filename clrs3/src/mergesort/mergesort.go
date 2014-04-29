// CLRS3 2.3.1
package mergesort

import (
	"math"
)

const (
	Infinity = math.MaxInt64
)

func merge(a []int64, p int, q int, r int) {
	n1 := q - p + 1
	n2 := r - q + 1
	left := make([]int64, n1)
	right := make([]int64, n2)
	for i := 0; i < n1-1; i++ {
		left[i] = a[p+i]
	}
	for j := 0; j < n2-1; j++ {
		right[j] = a[q+j]
	}
	left[n1-1] = Infinity
	right[n2-1] = Infinity
	i := 0
	j := 0
	for k := p; k < r; k++ {
		if left[i] < right[j] {
			a[k] = left[i]
			i += 1
		} else {
			a[k] = right[j]
			j += 1
		}
	}
}

func Sort(a []int64) {
}

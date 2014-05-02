// CLRS3 2.3.1
package mergesort

/* CLRS3 has 1-based indexing, but Go doesn't.
 * Here we're merging a[p:q] and a[q:r] with more idiomatic Go.
 */
func merge(a []int64, p int, q int, r int) {
	left := make([]int64, q-p)
	right := make([]int64, r-q)
	for i := 0; i < len(left); i++ {
		left[i] = a[p+i]
	}
	for j := 0; j < len(right); j++ {
		right[j] = a[q+j]
	}
	i := 0
	j := 0
	for k := p; k < r; k++ {
		if i >= len(left) {
			a[k] = right[j]
			j += 1
		} else if j >= len(right) {
			a[k] = left[i]
			i += 1
		} else if left[i] < right[j] {
			a[k] = left[i]
			i += 1
		} else {
			a[k] = right[j]
			j += 1
		}
	}
}

func sort(a []int64, p int, r int) {
	if r-p > 1 {
		q := (p + r) / 2
		sort(a, p, q)
		sort(a, q, r)
		merge(a, p, q, r)
	}
}

func Sort(a []int64) {
	sort(a, 0, len(a))
}

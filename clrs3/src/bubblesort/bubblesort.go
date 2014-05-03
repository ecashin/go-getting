// CLRS3 exercise 2.2
package bubblesort

func Sort(a []int64) {
	for i, _ := range a {
		for j := len(a) - 1; j > i; j-- {
			if a[j] < a[j-1] {
				n := a[j]
				a[j] = a[j-1]
				a[j-1] = n
			}
		}
	}
}

package main

import (
	"intarr"
)

func insSort(a []int64) {
	for j := 1; j < len(a); j++ {
		key := a[j]
		i := j - 1
		for i > -1 && a[i] > key {
			a[i+1] = a[i]
			i--
		}
		a[i+1] = key
	}
}

func main() {
	intarr.SortStdin(insSort)
}

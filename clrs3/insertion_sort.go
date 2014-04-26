package main

import (
	"fmt"
	"intarr"
)

func insSort(a []int64) {
	for i := 1; i < len(a); i++ {
		fmt.Print(a[i])
	}
}

func main() {
	intarr.SortStdin(insSort)
}

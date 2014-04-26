package intarr

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func SortStdin(sorter func([]int64)) {
	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadBytes('\n')
	a := []int64{}
	for err == nil {
		flds := strings.Fields(string(line))
		for _, f := range flds {
			i, err := strconv.ParseInt(f, 0, 64)
			if err != nil {
				panic(err)
			}
			a = append(a, i)
		}
		line, err = in.ReadBytes('\n')
	}
	fmt.Printf("%v", a)
}

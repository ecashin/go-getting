package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadBytes('\n')
	a := []string{}
	for err == nil {
		flds := strings.Fields(string(line))
		for _, i := range flds {
			a = append(a, i)
		}
		line, err = in.ReadBytes('\n')
	}
	fmt.Printf("%q", a)
}

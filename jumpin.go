// jumpin - compiler for moving text
//
// Now it's a skeleton.  The plan is to parse the shortest text
// that is between "@{" and "}" on the same line.
//
// go-getting$ echo "1@{2}3@{4}5" | ~/hg/go/bin/go run jumpin.go
// hi
// 1JMP(2)3JMP(4)REMAINING:5
// go-getting$ 

package main

import (
	"fmt"
	"bufio"
	"os"
	"regexp"
)

func jmp(o *os.File, t []byte) {
	fmt.Fprint(o, "JMP(", string(t), ")")
}

func main() {
	fmt.Println("hi")
	jre := regexp.MustCompile("(.*?)@{(.*?)}")
	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadBytes('\n')
	for err == nil {
		m := jre.FindAllSubmatch(line, -1)
		n := 0
		for _, g := range m {
			if len(g) != 3 {
				panic("derp!")
			}
			n += len(g[0])
			fmt.Print(string(g[1]))
			jmp(os.Stdout, g[2])
		}
		fmt.Print("REMAINING:", string(line[n:]))
		line, err = in.ReadBytes('\n')
	}
}

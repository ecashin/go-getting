// ecashin@montgomery:~/git/go-getting$ echo a b b c c c | go run wc.go
// a 1
// b 2
// c 3
// ecashin@montgomery:~/git/go-getting$ 

package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(bufio.ScanWords)

	counts := make(map[string]int)

	for scanner.Scan() {
		counts[scanner.Text()]++
	}
	for key, value := range counts {
		fmt.Println(key, value)
	}
}

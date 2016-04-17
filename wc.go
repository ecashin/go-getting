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

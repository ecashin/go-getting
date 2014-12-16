package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("hi", len(Bin))
	f, err := os.OpenFile("/tmp/selfexe-payload", os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	f.WriteString(Bin)
	f.Close()
}

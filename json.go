// don't know why this prints "{}" for me

package main

import (
	"encoding/json"
	"os"
)

func main() {
	type TData struct {
		a string
		b int
	}
	td := TData {
		a: "the letter a",
		b: 23,
	}
	j, err := json.Marshal(td)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(j)
}

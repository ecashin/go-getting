// You only get fields that have capitalized names in the JSON.

package main

import (
	"encoding/json"
	"os"
)

func main() {
	type TDataNoWorkie struct {
		a string
		b int
	}
	type TData struct {
		A string
		B int
		c string
	}
	td := TDataNoWorkie{
		a: "the letter a",
		b: 23,
	}
	tdw := TData{
		A: "the letter a",
		B: 23,
		c: "will not appear",
	}
	j, err := json.Marshal(td)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(j)

	j, err = json.Marshal(tdw)
	if err != nil {
		panic(err)
	}
	os.Stdout.Write(j)
}

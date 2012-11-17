package main

import (
	"fmt"
	"container/list"
)

func forEach(p *list.List, action func(s string) bool) {
	for e := p.Front(); e != nil; e = e.Next() {
		s := e.Value.(string)
		if ! action(s) {
			break
		}
	}
}

func main() {
	p := list.New()
	p.PushBack("a")
	p.PushBack("b")
	p.PushBack("c")
	p.PushBack("d")
	i := 0		// note: local variable
	forEach(p, func(s string) bool {
		i++	// closure on i affects main's i
		fmt.Printf("callback saw %s\n", s)
		if s == "c" {
			return false
		}
		return true
	})
	fmt.Printf("i: %d\n", i)	// prints "i: 3\n"
}

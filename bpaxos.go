// First inklings of basic Paxos implementation in Go
// see:
//   http://en.wikipedia.org/wiki/Paxos_%28computer_science%29
//   http://research.microsoft.com/en-us/um/people/lamport/pubs/pubs.html#lamport-paxos
//

package main

import (
	"fmt"
)

type proposer struct {
	num int64
}

type acceptor struct {
	biggest int64
}

type val struct {
	v string
	num int64
	rsp chan bool
}

func (p *proposer) propose(n int, propose, promise chan int64, value chan val) {
	p.num++
	for i := 0; i < n; i++ {
		propose <- p.num
	}
	for i := 0; i < n; i++ {
		if <-promise != p.num {
			panic("nack")
		}
	}
	ok := make(chan bool)
	v := val{"the word of the day is shiny", p.num, ok}
	for i := 0; i < n; i++ {
		value <- v
		if ! <- ok {
			panic("not accepted")
		}
	}
	fmt.Println("consensus:", v.v)
}

func (a *acceptor) accept(propose, promise chan int64, value chan val) {
	for {
		select {
		case n := <- propose:
			if n > a.biggest {
				a.biggest = n
				promise <- n
			} else {
				promise <- -1
			}
		case v := <- value:
			if a.biggest <= v.num {
				fmt.Println("accepting ", v.num)
				v.rsp <- true
			} else {
				v.rsp <- false
			}
		}
	}	
}

func main() {
	p := &proposer{0}
	a := &acceptor{-1}
	propose := make(chan int64)
	promise := make(chan int64)
	value := make(chan val)
	go a.accept(propose, promise, value)
	p.propose(1, propose, promise, value)
}

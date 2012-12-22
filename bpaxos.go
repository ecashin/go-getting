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
	name string
	num int64
}

type acceptor struct {
	biggest int64
	val value
}

type value *string

type proposal struct {
	num int64
	val value
}

type sendProposal struct {
	proposal
	c chan proposal
}

func (p *proposer) propose(n int, acceptors chan sendProposal) bool {
	p.num++
	c := make(chan proposal)
	for i := 0; i < n; i++ {
		acceptors <- sendProposal{proposal{p.num, nil}, c}
	}
	var v value

	// XXX needs timeout for acceptor failures
	for i := 0; i < n; i++ {
		rsp := <-c
		if rsp.num != p.num {
			p.num = rsp.num
			return false
		}
		if rsp.val != nil {
			v = rsp.val
		}
	}
	if v == nil {
		s := "my name is " + p.name
		v = &s
	}
	for i := 0; i < n; i++ {
		// attempt to set value
		acceptors <- sendProposal{proposal{p.num, v}, c}
	}
	// XXX needs timeout for acceptor failures
	for i := 0; i < n; i++ {
		rsp := <-c
		if rsp.num != p.num {
			p.num = rsp.num
			return false
		}
		if rsp.val != v {
			return false
		}
	}
	close(acceptors)
	fmt.Println("consensus:", *v)
	return true
}

func (a *acceptor) show(cmd sendProposal) {
	t := "proposed"
	if cmd.val != nil {
		t = "set"
	}
	fmt.Println(t, a.biggest, a.val)
}

func (a *acceptor) accept(proposers chan sendProposal) {
	for cmd := range proposers {
		if cmd.val == nil {
			if cmd.num > a.biggest {
				a.biggest = cmd.num
			}
		} else {
			if cmd.num >= a.biggest && a.val == nil {
				a.val = cmd.val
			}
		}
		a.show(cmd)
		cmd.c <- proposal{a.biggest, a.val}
	}	
}

func main() {
	pa := &proposer{"alice", 0}
	// pb := &proposer{"bob", 0}
	a := &acceptor{-1, nil}
	cmdc := make(chan sendProposal)
	go a.accept(cmdc)
	fmt.Println("did it work? ", pa.propose(1, cmdc))
}

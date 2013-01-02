// First inklings of basic Paxos implementation in Go
// see:
//   http://en.wikipedia.org/wiki/Paxos_%28computer_science%29
//   http://research.microsoft.com/en-us/um/people/lamport/pubs/pubs.html#lamport-paxos
//

package main

import (
	"fmt"
	"log"
	"flag"
)

type proposer struct {
	name string
	num int64
	success bool
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

type proposalMsg struct {
	proposal
	sender string
	c chan proposal
}

func (p *proposer) propose(n int, acceptors chan proposalMsg, exit chan bool) {
	defer func() {
		exit <- p.success
	}()
	p.num++
	c := make(chan proposal)
	for i := 0; i < n; i++ {
		// try out this proposal number
		log.Printf("%s sending proposed number %d iter %d\n",
			p.name, p.num, i)
		acceptors <- proposalMsg{proposal{p.num, nil}, p.name, c}
	}
	var v value

	abort := false
	// XXX needs timeout for acceptor failures
tmp := p.num
	for i := 0; i < n; i++ {
		log.Printf("%s reading rsp to proposed number %d iter %d\n",
			p.name, tmp, i)
		rsp := <-c
		if rsp.num != p.num {
			p.num = rsp.num
			p.success = false
			abort = true
		}
		if rsp.val != nil {
			v = rsp.val
		}
	}
	if abort {
		log.Printf("%s aborting after proposed number\n", p.name)
		p.success = false
		return		
	}
	if v == nil {
		s := fmt.Sprintf("v%d my name is %s", p.num, p.name)
		v = &s
	}
	for i := 0; i < n; i++ {
		// attempt to set value
		log.Printf("%s setting val for proposed number %d iter %d\n",
			p.name, p.num, i)
		acceptors <- proposalMsg{proposal{p.num, v}, p.name, c}
	}

	// XXX needs timeout for acceptor failures
	for i := 0; i < n; i++ {
		log.Printf("%s reading rsp after setting val for proposed number %d iter %d\n",
			p.name, p.num, i)
		rsp := <-c
		if rsp.num != p.num {
			p.num = rsp.num
			p.success = false
			abort = true
		}
		if rsp.val != v {
			p.success = false
			abort = true
		}
	}
	p.success = !abort
	log.Printf("%s exiting with success(%v)\n",
		p.name, p.success)
}

func (a *acceptor) show(cmd proposalMsg) {
	t := ""
	if cmd.val != nil {
		t += "set value \"" + *cmd.val + "\""
	} else {
		t += "propose " + string(cmd.num)
	}
	// XXX changing the line below to "log.Printf" reveals deadlock
	fmt.Printf("acceptor saw %s %s responding with biggest:%d val:%v\n",
		 cmd.sender, t, a.biggest, a.val)
}

func (a *acceptor) accept(proposers chan proposalMsg) {
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

var nProds int
var nAccs int
func init() {
	flag.IntVar(&nProds, "p", 1, "specify number of proposers")
	flag.IntVar(&nAccs, "a", 3, "specify number of acceptors")
}
func main() {
	flag.Parse()
	fmt.Println(nProds, nAccs)
	pexitc := make(chan bool)	// for the proposers to signal exit
	p := make([]*proposer, nProds)
	for i := 0; i < nProds; i++ {
		p[i] = &proposer{fmt.Sprintf("proposer%d", i), 0, false}
	}
	cmdc := make(chan proposalMsg)
	for i := 0; i < nAccs; i++ {
		a := &acceptor{-1, nil}
		go a.accept(cmdc)
	}
	for i := 0; i < nProds; i++ {
		go p[i].propose(nAccs, cmdc, pexitc)
	}
	for i := 0; i < nProds; i++ {
		<-pexitc
	}
	for i := 0; i < nProds; i++ {
		fmt.Printf("%s got consensus? %v\n", p[i].name, p[i].success)
	}
	close(cmdc)
}

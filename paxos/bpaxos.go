// First inklings of basic Paxos implementation in Go
// see:
//   http://en.wikipedia.org/wiki/Paxos_%28computer_science%29
//   http://research.microsoft.com/en-us/um/people/lamport/pubs/pubs.html#lamport-paxos
// I started this based on wikipedia but needed Lamport's
// "Paxos Made Simple" to really nail down the details, like
// the fact that propoers use numbers from disjoint sets.
//
//   http://research.microsoft.com/en-us/um/people/lamport/pubs/pubs.html#paxos-simple
//
// TODO:
// * make all nodes peers, able to assume any role
// * timeouts for unresponsive participants
// * switch to UDP-based networking and distributed implementation

package main

import (
	"fmt"
	"log"
	"flag"
)

const NMaxProposers = 10

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

func (p *proposer) exceed(n int64) {
	// different producer should not use my numbers
	if p.num == n {
		panic("derp")
	}
	for p.num < n {
		p.num += NMaxProposers
	}
}

func (p *proposer) propose(n int, acceptors chan proposalMsg, exit chan bool) {
	defer func() {
		exit <- p.success
	}()
	c := make(chan proposal)
	for i := 0; i < n; i++ {
		// try out this proposal number
		log.Printf("%s sending proposed number %d iter %d\n",
			p.name, p.num, i)
		acceptors <- proposalMsg{proposal{p.num, nil}, p.name, c}
	}
	var v value

	abort := false
	orig_pnum := p.num
	// "Paxos Made Simple", Phase 2. (a), says
	// that proposer uses value with highest number
	// from responses to "prepare" messages.
	maxRspNum := int64(-1)
	for i := 0; i < n; i++ {
		log.Printf("%s reading rsp to proposed number %d iter %d\n",
			p.name, orig_pnum, i)
		rsp := <-c
		if rsp.num != p.num {
			p.exceed(rsp.num)
			p.success = false
			abort = true
		}
		if rsp.val != nil && rsp.num > maxRspNum {
			maxRspNum = rsp.num
			v = rsp.val
		}
	}
	if abort {
		log.Printf("%s aborting after proposing number %d\n",
			p.name, orig_pnum)
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

	for i := 0; i < n; i++ {
		log.Printf("%s reading rsp after setting val for proposed number %d iter %d\n",
			p.name, p.num, i)
		rsp := <-c
		if rsp.num != p.num {
			p.exceed(rsp.num)
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
		t += fmt.Sprintf("propose %d", cmd.num)
	}
	// XXX changing the line below to "log.Printf" reveals deadlock
	fmt.Printf("acceptor saw %s %s; responding with biggest:%d val:%v\n",
		 cmd.sender, t, a.biggest, a.val)
}

func (a *acceptor) handleCmd(cmd proposalMsg) {
	if cmd.val == nil {
		// it's a Prepare message
		if cmd.num > a.biggest {
			a.biggest = cmd.num

			// When acceptor gets higher num
			// prepare, it responds with its current
			// value. The proposer sees all the
			// values in the prepare responses and
			// picks the highest numbered one. The
			// same acceptor will then accept
			// whatever the proposer says is the
			// value.  That implies the acceptor must
			// retain the previously accepted value
			// at this step.
		}
	} else {
		// Accept! message
		if cmd.num >= a.biggest {
			// for unreliable delivery of Accept!
			// messages with cmd.num > a.biggest,
			// proposer can keep trying for a quorum
			a.val = cmd.val
		}
	}
	a.show(cmd)
	cmd.c <- proposal{a.biggest, a.val}
}

func (a *acceptor) accept(proposers chan proposalMsg) {
	for cmd := range proposers {
		go a.handleCmd(cmd)
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
		p[i] = &proposer{fmt.Sprintf("proposer%d", i), int64(i), false}
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

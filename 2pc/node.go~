// clubpaxos.go
// This is the skeleton of a process that will be able to join
// or found a club, where the club can outlive any particular
// set of processes that are members.
//
// Club State
//
//   a list of members
//
// User Message Format Examples
//
//   node named "david" asks to join club "boodles":
//     join boodles david
//
//   node ask for club name
//     name
//
// Club Operations (not messages but stuff that happens)
//
//   paul add {name}	paul asks club to add member {name}
//   paul oust {name}	paul asks club to remove member {name}
//   paul toast {name}	raise a glass to health of a member
//
// Club Message Format Examples
//
//   paul, acting leader of "boodles" asks the club to add new
//   member "david with instance number 50 proposal number 1.
//     club boodles paul 50 1 add david
//     [gets accepts]
//
//   ron, first-time leader of "boodles" uses instance 34 of paxos to
//   ask the club to add new member "david", using proposal number
//   5 (presumably because someone else is trying to lead):
//     club boodles ron 34 5 propose
//     [gets promises]
//     club boodles ron 34 5 propose add david
//     [gets accepts]

package main

import (
	"flag"
	"log"
	"net"
	"strings"
	"time"
)

type ureq struct {
	f []string // fields of last user request
	t time.Time
}

type node struct {
	name     string // my name
	step     int64  // Paxos algorithm instance
	pseen    int64  // greatest proposal num seen for this instance
	lastReq  ureq   // last user request
	amLeader bool
}

func (p *node) nextProposal() int64 {
	rounds := p.pseen / maxNodes
	m := maxNodes*rounds + int64(myID)
	m += maxNodes
	p.pseen = m
	return m
}

func (p *node) askClub(s string) {
	if p.amLeader {
		log.Printf("club %s %d %d %d %s",
			myClub, myID, p.step, p.nextProposal(), s)
	} else {
		panic("XXX implement new leader")
	}
}

func (p *node) handleJoin(f []string) {
	if len(f) < 3 {
		log.Printf("%s ignoring malformed join\n", p.name)
		return
	}
	if myClub != f[1] {
		return
	}
	if p.amLeader {
		p.askClub("add " + f[2])
	} else {
		p.lastReq = ureq{f, time.Now()}
	}
}

func (p *node) serve(c chan string) {
	// see go/src/pkg/net/protoconn_test.go
	la, err := net.ResolveUDPAddr("udp4", "127.0.0.1:9999")
	if err != nil {
		panic(err)
	}
	conn, err := net.ListenUDP("udp4", la)
	if err != nil {
		panic(err)
	}
	buf := make([]byte, 9999)
	for {
		n, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			panic(err)
		}
		s := string(buf[:n])
		log.Printf("%s says %s", raddr, s)
		c <- s
	}
	c <- "EOF"
}

const maxNodes = 4

var I node
var myClub string
var myID int

const noClub = "(no club)"

func init() {
	flag.IntVar(&myID, "i", -1,
		"the unique integer ID of this participant")
	flag.StringVar(&myClub, "c", noClub,
		"the club this participant will found")
}
func main() {
	flag.Parse()
	if myID < 0 {
		panic("no ID provided")
	} else if myID >= maxNodes {
		panic("max ID is " + string(maxNodes-1))
	}
	if myClub != noClub {
		log.Printf("%d founding club %s", myID, myClub)
		I.amLeader = true
	}

	c := make(chan string, 5)
	go I.serve(c)
	log.Printf("%d started", myID)
	for {
		s := <-c
		if s == "EOF" {
			break
		}
		f := strings.Fields(s)
		switch strings.ToLower(f[0]) {
		case "join":
			I.handleJoin(f)
		}
	}
}

// clubpaxos.go
// This is the skeleton of a process that will be able to join
// or found a club, where the club can outlive any particular
// set of processes that are members.

package main

import (
	"flag"
	"log"
	"net"
)

type node struct {
	step     int64
	psent    int64 // last proposal sent
	precv    int64 // last proposal seen
	amLeader bool
}

func (n *node) nextProposal() int64 {
	if n.psent < n.precv {
		rounds := n.precv / maxNodes
		n.psent = maxNodes*rounds + int64(myID)
	}
	n.psent += maxNodes
	return n.psent
}

func (n *node) serve(c chan string) {
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
			log.Print(err)
			break
		}
		c <- raddr.String() + " says " + string(buf[:n])
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

	c := make(chan string)
	go I.serve(c)
	log.Printf("%d started", myID)
	for i := 0; i < 20; i++ {
		I.precv = int64(i)
		log.Printf("proposing with %d", I.nextProposal())
	}
	for {
		s := <-c
		if s == "EOF" {
			break
		}
		log.Print(s)
	}
}

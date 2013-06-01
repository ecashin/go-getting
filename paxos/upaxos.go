// upaxos.go - UDP-based Paxos participant implementation
//
//	The peers are expected to be supplied line-by-line
//	on standard input, in the form: a.b.c.d:p, an IP
//	address a.b.c.d and port p.
//
// example usage:
// ecashin@atala paxos$ go run upaxos.go < upaxos-peers.txt
//
// ecashin@atala ~$ echo accept this | nc -u 127.0.0.1 9876
//
// DESIGN
//
// The main thread handles starts a goroutine:
//
//   listener: receives messages
//
// ... which copies each message into multiple channels, one
// for each role that acts on received messages.  Goroutines
// for each such role ignore or act on the messages as appropriate.
//
//   leader:   handles Request, NACK, Promise, Accept;
//   	       sends Propose, Assign, Response
//
//   acceptor: handles Propose, Assign;
//   	       sends NACK, Promise, Accept
//
//   learner:  notes observed quorums;
//             can respond to requests about previous 
//             paxos instances (history)
//
// There's an interplay between leading and accepting.  For example,
// if I see a new request but expect a peer to take the lead, I should
// timeout and take the lead myself (delayed according to my ID, to
// avoid racing with other peers), if I don't see any "propose"
// message.  So the state machine design is attractive.
//
// However, a pure state machine design is awkward because the real
// state of the program is an N-tuple for the N concurrently running
// roles, and the combinations mean there are very many states.
//
// Using goroutines should result in more understandable code.

package main

import (
	"bufio"
	"container/list"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var myAddr string
var myID int = -1

func group() []string {
	grp := list.New()
	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadSlice('\n')
	i := 0
	for err == nil {
		fields := strings.Fields(string(line))
		if len(fields) > 0 {
			s := fields[0]
			grp.PushBack(s)
			if fields[0] == myAddr {
				myID = i
				s = fmt.Sprintf("  me: %s %d", s, i)
			} else {
				s = "peer: " + s
			}
			i += 1
			log.Print(s)
		}
		line, err = in.ReadSlice('\n')
	}
	g := make([]string, i)
	i = 0
	for e := grp.Front(); e != nil; e = e.Next() {
		g[i] = e.Value.(string)
		i++
	}
	if myID == -1 {
		log.Panic("could not determine my proposal number set")
	}
	return g
}

type Msg struct {
	s string
	conn *net.UDPConn
	raddr *net.UDPAddr
}

type Req struct {
	proposed bool
	accepts map[string]bool
	val string
}
func req(f []string) Req {
	if len(f) != 2 || f[0] != "Request" {
		panic("called req with bad string")
	}
	return Req{false, make(map[string] bool), f[1]}
}

type Nack struct {
	instance, pnum int64
}
func nack(f []string) Nack {
	if len(f) != 3 || f[0] != "NACK" {
		panic("called nack on bad string")
	}
	i, err := strconv.ParseInt(f[1], 0, 64)
	if err != nil {
		panic(err)
	}
	p, err := strconv.ParseInt(f[2], 0, 64)
	if err != nil {
		panic(err)
	}
	return Nack{i, p}
}

type leader struct {
	g []string	// the group of Paxos participants
	lastp int64
	rq *list.List
}
func (ld *leader) nextp() {
	ld.lastp += int64(len(ld.g))
}
func (ld *leader) propose(r Req) {
	log.Printf("propose Req{%v, %v, %v}", r.proposed, r.accepts, r.val)
}
func lead(c chan Msg, g []string) {
	ld := leader{g, int64(myID), list.New()}
	for {
		select {
		case m := <- c:
			log.Printf("leader got \"%s\"", m.s)
			f := strings.Fields(m.s)
			if len(f) == 0 {
				continue
			}
			switch f[0] {
			case "Request":
				r := req(f)
				ld.rq.PushBack(r)
				r = ld.rq.Front().Value.(Req)
				if !r.proposed {
					ld.propose(r)
				}
			case "NACK":
				nk := nack(f)
				log.Print(nk.instance)
			}
		case <- time.After(50 * time.Millisecond):
			if ld.rq.Front() != nil {
				log.Print("service request")
			}
		}
	}
}
func accept(c chan Msg) {
	for m := range c {
		log.Printf("acceptor got \"%s\"", m.s)
	}
}
func learn(c chan Msg, nGrp int) {
	for m := range c {
		log.Printf("learner got \"%s\"", m.s)
	}
}

func listen(chans []chan Msg) {
	la, err := net.ResolveUDPAddr("udp4", myAddr)
	if err != nil {
		log.Panic(err)
	}
	conn, err := net.ListenUDP("udp4", la)
	if err != nil {
		log.Panic(err)
	}
	buf := make([]byte, 9999)
	for {
		n, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Panic(err)
		}
		s := string(buf[:n])
		if len(strings.Fields(s)) == 0 {
			continue
		}
		for _, c := range chans {
			c <- Msg{s, conn, raddr}
		}
	}
}

func send(s string, conn *net.UDPConn, raddr *net.UDPAddr) {
	_, err := conn.WriteToUDP([]byte(s), raddr)
	if err != nil {
		log.Panic(err)
	}
}

func init() {
	flag.StringVar(&myAddr, "a", "127.0.0.1:9876",
		"IP and port for this process")
}
func main() {
	flag.Parse()
	log.Print("upaxos started at ", myAddr)
	defer log.Print("upaxos ending")

	g := group()
	for i := 0; i < len(g); i++ {
		log.Print(g[i])
	}
	leadc := make(chan Msg)
	acceptc := make(chan Msg)
	learnc := make(chan Msg)
	mainc := make(chan Msg)
	go lead(leadc, g)
	go accept(acceptc)
	go learn(learnc, len(g))
	go listen([]chan Msg{leadc, acceptc, learnc, mainc})
loop:
	for m := range mainc {
		flds := strings.Fields(m.s)
		if len(flds) > 0 {
			switch flds[0] {
			case "quit": fallthrough
			case "exit": fallthrough
			case "bye":
				log.Print("exiting")
				break loop
			}
		}
	}
}

// upaxos.go - UDP-based Paxos participant implementation
//
// Features for short term:
//
//   * multiple states (AKA Multi-Paxos or "Parilament")
//
//   * history is fast-readable, requiring no consensus instance
//
//   * NACK messages allow leaders to operate efficiently
//
//   * leaders back off deterministically to enhance liveness
//
// Features to do next:
//
//   * persistent storage needed for recovery ("logs")
//
//   * recovery - participant starts up, reads logs, and participates
//
//   * support "Join N" command, where N is the last instance
//       learned by the candidate node.  A learner responds with
//	 the value a quorum accepted for instance N if it's historical.
//	 In that case, the candidate can try "Join N+1" if it learns
//	 the value for N.
//
//	 Or, if there is no quorum yet for instance N, a leader
//	 attempts to propose that the group be enlarged.  When
//	 the candidate sees consensus on its own membership in
//	 the group, it is up to date and fully participating.
//
//	 Nobody joins with an incomplete history, so the current
//	 group can always answer questions about past states.
//
//	 This will be the R_1 reconfiguration scheme described in
//	 Lamport's Reconfiguration Tutorial.
//
// Features for Someday or Never:
//
//   * history compaction in learner
//
//	Once the learner knows a quorum has accepted a value,
//	it can forget all the extra information about who accepted
//	what with what proposal number.
//
//   * Concurrent consensus instances
//
// The peers are expected to be supplied line-by-line
// on standard input, in the form: a.b.c.d:p, an IP
// address a.b.c.d and port p.
//
// example usage:
// ecashin@atala paxos$ go run upaxos.go < upaxos-peers.txt
//
// ecashin@atala ~$ echo Request 0 do stuff | nc -u 127.0.0.1 9876
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
//   	       sends Propose, Fix, Response
//
//   acceptor: handles Propose, Fix;
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
	f []string	// whitespace-separated message fields
	conn *net.UDPConn
	raddr *net.UDPAddr
}

func mustStrtoll(s string) (n int64) {
	n, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		panic(err)
	}
	return
}

// unlike Wikipedia, it's instance first
func instanceProposal(f []string) (p, i int64) {
	i = mustStrtoll(f[1])
	p = mustStrtoll(f[2])
	return
}

// message format:
// Request I V
// I	consensus instance (zero for new state)
// V	the value the client wants to set (ignored for lookup)
type Req struct {
	i int64	// 0 for new instance
	v string // ignored for history query
}
func newReq(f []string) Req {
	if len(f) < 2 || f[0] != "Request" {
		panic("called newReq with bad string")
	}
	i := mustStrtoll(f[1])
	s := ""
	if len(f) > 2 {
		s = strings.Join(f[2:], " ")
	}
	return Req{i, s}
}

// message format:
// Propose I P
// I	instance
// P	the proposal number the leader is attempting to use
type Propose struct {
	instance, p int64
}
func newPropose(f []string) Propose {
	if len(f) < 3 || f[0] != "Propose" {
		log.Panic("called newPropose with bad string")
	}
	p, i := instanceProposal(f)
	return Propose{i, p}
}

// message format:
// Promise I A [B V]
// where...
// I	instance
// A	the minimum proposal number sender will accept
// B	the proposal number associated with previously accepted value
// V	the previously accepted value
type Promise struct {
	// required fields
	minp, instance int64

	// optional fields, absent if no prior accepted value
	valp int64	// proposal number associated with the accepted value
	value *string	// the previously accepted value, nil if none accepted
}
func newPromise(f []string) Promise {
	if len(f) < 3 || f[0] != "Promise" {
		log.Panic("called newPromise with bad string")
	}
	p, i := instanceProposal(f)
	vp := int64(0)
	var v *string
	if len(f) > 3 {
		vp = mustStrtoll(f[3])
		s := ""
		if len(f) > 4 {
			s = strings.Join(f[4:], " ")
		}
		v = &s
	}
	return Promise{p, i, vp, v}
}

// message format:
// I	consensus instance number
// P	proposal number
// V	value accepted
type Accept struct {
	i, p int64
	v string
}
func newAccept(f []string) Accept {
	if len(f) < 3 || f[0] != "Accept" {
		panic("called newAccept with bad string")
	}
	p, i := instanceProposal(f)	
	return Accept{i, p, strings.Join(f[3:], " ")}
}

// message format:
// I	consensus instance
// P	minimum acceptable proposal number
type Nack struct {
	i, p int64
}
func newNack(f []string) Nack {
	if len(f) != 3 || f[0] != "NACK" {
		panic("called newNack on bad string")
	}
	i, p := instanceProposal(f)
	return Nack{i, p}
}

const maxReqQ = 10	// max 10 queued requests

// XXXtodo:
//   * make sure v and vp are reset on new consensus instance
func lead(c chan Msg, g []string) {
	instance := int64(0)	// consensus instance
	lastp := int64(myID)	// proposal number last sent
	rq := list.New()	// queued requests
	nrq := 0		// number of queued requests
	var r *Req		// client request in progress
	var v *string	// value to fix; nil means can fix client value
	vp := int64(-1)	// proposal number associated with v
	npromise := 0	// number of promises received for r
	catchup := func(p int64) int64 {
		npromise = 0
		n := int64(len(g))
		p /= n
		p++
		return p * n + int64(myID)
	}

	everybody := make([]string, len(g)+1)
	for i, _ := range g {
		everybody[i] = g[i]
	}
	everybody[len(g)] = myAddr

	propose := func() {
		s := fmt.Sprintf("propose I:%d P:%d V:%s",
			instance, lastp, r.v)
		for _, ra := range everybody {
			send(s, nil, ra)
		}
	}
	for {
		select {
		case m := <- c:
			log.Printf("leader got \"%s\"", m.f[0])
			switch m.f[0] {
			case "Promise":
				if r == nil {
					log.Print("ignoring Promise--no Req in progress")
					continue
				}
				p := newPromise(m.f)
				if p.instance != instance {
					instance = p.instance
					lastp = catchup(p.minp)
					log.Printf("instance mismatch: %d vs %d",
						p.instance, instance)
					continue
				} else if p.minp != lastp {
					lastp = catchup(p.minp)
					log.Printf("proposal mismatch (%d %d) try again", p.minp, lastp)
					continue
				}
				if p.value != nil {
					if p.valp > vp {
						v = p.value
						vp = p.valp
					}
				}
				npromise++
				log.Printf("got promise from %s\n", m.raddr)
				if npromise > len(g)/2 {
					log.Print("send Fix message", v)
					// XXXtodo: send Fix message
				}
			case "Accept":
				a := newAccept(m.f)
				log.Printf("got accept %v", a)
				// XXXtodo: record accept
				// XXXtodo:
				//   if a quorum has accepted,
				//     dequeue next req
				//     start next consensus instance
			case "Request":
				newr := newReq(m.f)
				if r == nil {
					r = &newr
					propose()
				} else if nrq < maxReqQ {
					rq.PushBack(newr)
					nrq++
				} else {
					log.Print("send BUSY to client")
				}
			case "NACK":
				nk := newNack(m.f)
				log.Printf("NACK for instance: %d", nk.i)
				// XXXtodo: catchup and re-propose
			}
		case <- time.After(50 * time.Millisecond):
			if rq.Front() != nil {
				log.Print("service request")
			}
		}
	}
}
func accept(c chan Msg) {
	// per-instance record of minimum proposal number we can accept
	minp := make(map[int64]int64)
	for m := range c {
		log.Printf("acceptor got \"%s\"", m.f[0])
		switch m.f[0] {
		case "Propose":
			p := newPropose(m.f)
			min, present := minp[p.instance]
			s := ""
			if !present {
				minp[p.instance] = p.p
				s = fmt.Sprintf("Promise %d %d", p.instance, p.p)
			} else if p.p < min {
				s = fmt.Sprintf("NACK %d %d", p.instance, minp)
			} else {
				// XXXtodo: include previously accepted value
			}
			m.conn.Write([]byte(s))
		case "Fix":
			log.Print("received fix")
			// XXXtodo: send NACK or record value and send Accept
		}
	}
}

// The Accepts for a given consensus instance
type AcceptRecord struct {
	h map[string]string	// accepted values by remote addr
	q map[string]int	// count of hosts by value accepted
}
func newAcceptRecord() AcceptRecord {
	return AcceptRecord{
		make(map[string]string),
		make(map[string]int),
	}
}
func learn(c chan Msg, g []string) {
	history := make(map[int64]AcceptRecord)
	for m := range c {
		switch m.f[0] {
		case "Accept":
			a := newAccept(m.f)
			if _, ok := history[a.i]; !ok {
				history[a.i] = newAcceptRecord()
			}
			ar := history[a.i]
			oldv, wasThere := ar.h[m.raddr.String()]
			ar.h[m.raddr.String()] = a.v
			if wasThere {
				ar.q[oldv] -= 1
			}
			ar.q[a.v] += 1
			log.Printf("learner got \"%s\" from %s", a.v, m.raddr)
		case "Request":
			r := newReq(m.f)
			if ar, present := history[r.i]; present {
				for v, n := range ar.q {
					if n > len(g)/2 {
						s := fmt.Sprintf("quorum %i: %s",
							r.i, v)
						m.conn.Write([]byte(s))
					}
				}
			}
		}
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
		f := strings.Fields(s)
		if len(f) == 0 {
			continue
		}
		for _, c := range chans {
			c <- Msg{f, conn, raddr}
		}
	}
}

func send(s string, conn *net.UDPConn, ra string) { //raddr *net.UDPAddr) {
	log.Printf("sending to %s: %s", ra, s)
	raddr, err := net.ResolveUDPAddr("udp", ra)
	if err != nil {
		log.Panic(err)
	}
	close := false
	if conn == nil {
		conn, err = net.DialUDP("udp", nil, raddr)
		if err != nil {
			log.Panic(err)
		}
		close = true
	}
	_, err = conn.Write([]byte(s))
	if err != nil {
		log.Panic(err)
	}
	if close {
		conn.Close()
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
	go learn(learnc, g)
	go listen([]chan Msg{leadc, acceptc, learnc, mainc})
loop:
	for m := range mainc {
		if len(m.f) > 0 {
			switch m.f[0] {
			case "quit": fallthrough
			case "exit": fallthrough
			case "bye":
				log.Print("exiting")
				break loop
			}
		}
	}
}

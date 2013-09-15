// upaxos.go - unreliable-broadcast-based Paxos participant
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
// example usage:
// ecashin@atala paxos$ sudo go run upaxos.go -n 3 -i 0 &
// ecashin@atala paxos$ sudo go run upaxos.go -n 3 -i 1 &
// ecashin@atala paxos$ sudo go run upaxos.go -n 3 -i 2 &
//
// ecashin@atala ~$ echo Request 0 do stuff | \
//	sudo go run iptest-send.go -a 127.0.0.1 -p 253
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
//   	       sends Propose, Write, Written
//
//   acceptor: handles Propose, Write;
//   	       sends NACK, Promise, Accept
//
//   learner:  notes observed quorums;
//             can respond to requests about previous 
//             paxos instances (history)
//
// The Request message is unusual in that its first field is not an
// identifier of a Paxos participant.  It is sent by a client, not
// necessarily a Paxos participant.
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
//
// Paxos over TCP would be complicated by the extra functionality
// of TCP that insulates software from the realities that Paxos is
// designed for, like failing networks and nodes.  Broadcast is
// especially nice, since several optimizations are allowed by
// snooping.

package main

import (
	"container/list"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const maxSends = 100	// in case things get out of hand, stop

var nSent int32
var myID int = -1
var nGroup int = -1
var receivers []chan Msg

type Msg struct {
	f []string	// whitespace-separated message fields
}

func mustStrtoll(s string) (n int64) {
	n, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		panic(err)
	}
	return
}

// unlike Wikipedia, it's instance first
func sipParse(f []string) (s, i, p int64) {
	s = mustStrtoll(f[0])
	i = mustStrtoll(f[2])
	p = mustStrtoll(f[3])
	return
}

// Request message format:
// I	consensus instance (zero for new state)
// V	the value the client wants to set (ignored for lookup)
type Req struct {
	i int64	// 0 for new instance
	v string // ignored for history query
}
func newReq(m Msg) Req {
	if len(m.f) < 2 || m.f[0] != "Request" {
		panic("called newReq with bad string")
	}
	i := mustStrtoll(m.f[1])
	s := ""
	if len(m.f) > 2 {
		s = strings.Join(m.f[2:], " ")
	}
	return Req{i, s}
}

// Propose message format:
// S	sender ID
// I	instance
// P	the proposal number the leader is attempting to use
type Propose struct {
	s, i, p int64
}
func newPropose(f []string) Propose {
	if len(f) < 4 || f[1] != "Propose" {
		log.Panic("called newPropose with bad string")
	}
	s, i, p := sipParse(f)
	return Propose{s, i, p}
}

// Promise message format is "S Promise I A [B V]", where...
// S	sender ID
// I	instance
// A	the minimum proposal number sender will accept
// B	the proposal number associated with previously accepted value
// V	the previously accepted value
type Promise struct {
	// required fields
	s, i, p int64

	// optional fields, absent if no prior accepted value
	vp int64	// proposal number associated with the accepted value
	v *string	// the previously accepted value, nil if none accepted
}
func logWritePromise(lf *log.Logger, i, p int64) {
	lf.Printf("promise %d %d", i, p)
}
func newPromise(f []string) Promise {
	if len(f) < 4 || f[1] != "Promise" {
		log.Panic("called newPromise with bad string")
	}
	src, i, p := sipParse(f)
	vp := int64(0)
	var v *string
	if len(f) > 4 {
		vp = mustStrtoll(f[4])
		s := ""
		if len(f) > 5 {
			s = strings.Join(f[5:], " ")
		}
		v = &s
	}
	return Promise{src, i, p, vp, v}
}

// Accept message format:
// S	sender ID
// I	consensus instance number
// P	proposal number
// V	value accepted
type Accept struct {
	s, i, p int64
	v string
}
func logWriteAccept(lf *log.Logger, i, p int64, v string) {
	lf.Printf("accept %d %d %s", i, p, v)
}
func newAccept(f []string) Accept {
	if len(f) < 4 || f[1] != "Accept" {
		panic("called newAccept with bad string")
	}
	src, i, p := sipParse(f)
	s := ""
	if len(f) > 4 {
		s = strings.Join(f[4:], " ")
	}
	return Accept{src, i, p, s}
}

// message format:
// S	sender ID
// I	consensus instance
// P	minimum acceptable proposal number
type Nack struct {
	s, i, p int64
}
func newNack(f []string) Nack {
	if len(f) != 4 || f[1] != "NACK" {
		panic("called newNack on bad string")
	}
	s, i, p := sipParse(f)
	return Nack{s, i, p}
}

// Write message format:
// S	sender ID
// I	consensus instance
// P	proposal number
// V	value
type Write struct {
	s, i, p int64
	v string
}
func newWrite(f []string) Write {
	if len(f) < 4 || f[1] != "Write" {
		panic("called newWrite with bad string")
	}
	s, i, p := sipParse(f)	
	return Write{s, i, p, strings.Join(f[4:], " ")}
}

const maxReqQ = 10	// max 10 queued requests

func lead(c chan Msg) {
	instance := int64(0)	// consensus instance leader is trying to use
	lastp := int64(myID)	// proposal number last sent
	rq := list.New()	// queued requests
	nrq := 0		// number of queued requests
	var r *Req		// client request in progress
	var v *string	// value to write
	vp := int64(-1)	// proposal number associated with v
	npromise := 0	// number of promises received for r
	naccepts := 0	// number of accepts received for r
	catchup := func(i, p int64) {
		if i != instance {
			v = nil
			vp = int64(-1)
		}
		instance = i
		npromise = 0
		naccepts = 0
		n := int64(nGroup)
		p /= n
		p++
		lastp = p * n + int64(myID)
	}
	nextInstance := func() {
		catchup(instance+1, 0)
	}

	for {
		select {
		case m := <- c:
			if len(m.f) < 2 {
				continue
			}
			if m.f[0] == "Request" {
				newr := newReq(m)
				if newr.v == "" {
					// let the learner answer this read
					continue
				}
				if r == nil {
					r = &newr
					s := fmt.Sprintf("%d Propose %d %d %s",
						myID, instance, lastp, r.v)
					go send(s)
				} else if nrq < maxReqQ {
					rq.PushBack(newr)
					nrq++
				} else {
					log.Print("send BUSY to client")
				}
				continue
			}
			switch m.f[1] {
			case "Promise":
				if r == nil {
					log.Print("ignoring Promise--no Req in progress")
					continue
				}
				p := newPromise(m.f)
				if p.i != instance {
					oldi := instance
					catchup(p.i, p.p)
					log.Printf("instance mismatch: %d => %d",
						oldi, p.i)
					continue
				} else if p.p < lastp {
					continue	// ignore lower-numbered proposals
				} else if p.p > lastp {
					catchup(p.i, p.p)	// snoop: like a NACK
					continue
				}
				if p.v != nil {
					if p.vp > vp {
						v = p.v
						vp = p.vp
					}
				}
				npromise++
				log.Printf("got promise from %d\n", p.s)
				if npromise > nGroup/2 {
					if v == nil {
						v = &r.v
					}
					s := fmt.Sprintf("%d Write %d %d %s",
						myID, instance, lastp, *v)
					go send(s)
				}
			case "Accept":
				if r == nil {
					log.Print("ignoring Accept with no Req in progress")
					continue
				}
				a := newAccept(m.f)
				if v == nil {
					log.Print("igoring Accept: v == nil")
					continue
				}
				if a.v != *v {
					log.Print("ignoring Accept: a.v != *v")
					log.Printf("a.v: %s", a.v)
					log.Printf(" *v: %s", *v)
					continue
				}
				if a.i != instance {
					log.Print("ignoring Accept: instance mismatch")
					continue
				}
				if a.p != lastp {
					log.Print("ignoring Accept: a.p != lastp")
					log.Printf("  a.p: %d", a.p)
					log.Printf("lastp: %d", lastp)
					continue
				}
				naccepts++
				if naccepts > nGroup/2 {
					if a.v == r.v {
						go send("OK")
						r = nil
						if rq.Front() != nil {
							e := rq.Front()
							r = e.Value.(*Req)
							rq.Remove(e)
						}
					}
					nextInstance()
				}
			case "NACK":
				nk := newNack(m.f)
				if nk.i > instance || nk.p > lastp {
					catchup(nk.i, nk.p)
				}
			}
		case <- time.After(30 * time.Second):
			log.Print("tick tock")	// demo
		}
	}
}
type Accepted struct {
	p int64
	v string
}
func accept(c chan Msg, lf *log.Logger) {
	// per-instance record of minimum proposal number we can accept
	minp := make(map[int64]int64)
	accepted := make(map[int64]Accepted)	// values by instance
	for m := range c {
		if len(m.f) < 2 {
			continue
		}
		switch m.f[1] {
		case "Propose":
			p := newPropose(m.f)
			s := fmt.Sprintf("%d ", myID)
			min, present := minp[p.i]
			if present && p.p < min {
				s += fmt.Sprintf("NACK %d %d", p.i, min)
			} else {
				minp[p.i] = p.p
				s += fmt.Sprintf("Promise %d %d", p.i, p.p)
				if va, there := accepted[p.i]; there {
					s += fmt.Sprintf(" %d %s", va.p, va.v)
				}
				logWritePromise(lf, p.i, p.p)
			}
			go send(s)
		case "Write":
			wr := newWrite(m.f)
			min, there := minp[wr.i]
			s := fmt.Sprintf("%d ", myID)
			if there && min > wr.p {
				log.Printf("acceptor with min %d ignoring Write %d %d %v",
					min, wr.i, wr.p, wr.v)
				s += fmt.Sprintf("NACK %d %d", wr.i, min)
			} else {
				s += fmt.Sprintf("Accept %d %d %s", wr.i, wr.p, wr.v)
				logWriteAccept(lf, wr.i, wr.p, wr.v)
				accepted[wr.i] = Accepted{wr.p, wr.v}
			}
			go send(s)
		}
	}
}

// The Accepts for a given consensus instance
// Storing the proposal number protects against out-of-order delivery
// of accept messages by the network.
type Accepts struct {
	v map[int64]string	// accepted values by participant (host) ID
	n map[string]int	// count of hosts by value accepted
	p map[int64]int64	// proposal number associated with value accepted by given host
}
func newAccepts() Accepts {
	return Accepts {
		make(map[int64]string),
		make(map[string]int),
		make(map[int64]int64),
	}
}
func learn(c chan Msg) {
	history := make(map[int64]Accepts)
	written := make(map[int64]string)	// quorum-accepted value by instance
	for m := range c {
		if len(m.f) < 2 {
			continue
		}
		if m.f[0] == "Request" {
			r := newReq(m)
			if as, present := history[r.i]; present {
				for v, n := range as.n {
					if n > nGroup/2 {
						s := fmt.Sprintf("%d OK %d %s",
							myID, r.i, v)
						go send(s)
						break
					}
				}
			}
			continue
		}
		switch m.f[1] {
		case "Accept":
			a := newAccept(m.f)
			if _, ok := written[a.i]; ok {
				log.Printf("ignoring Accept for written instance %d",
					a.i)
				continue
			}
			if _, ok := history[a.i]; !ok {
				history[a.i] = newAccepts()
			}
			as := history[a.i]
			oldv, wasThere := as.v[a.s]
			if wasThere && a.p < as.p[a.s] {
				continue	// ignore old Accept
			}
			as.v[a.s] = a.v
			as.p[a.s] = a.p
			if wasThere {
				as.n[oldv] -= 1
			}
			as.n[a.v] += 1
			log.Printf("learner got \"%s\" from %d, for %d accepts",
				a.v, a.s, as.n[a.v])
			if as.n[a.v] > nGroup/2 {
				written[a.i] = a.v
			}
		}
	}
}

func listen(conn *net.IPConn) {
	buf := make([]byte, 9999)
	for {
		n, _, err := conn.ReadFromIP(buf)
		if err != nil {
			log.Panic(err)
		}
		s := string(buf[:n])
		log.Printf("RECV %s", s)
		f := strings.Fields(s)
		if len(f) == 0 {
			log.Print("skipping zero-field message")
			continue
		}
		for _, c := range receivers {
			c <- Msg{f}
		}
	}
}

var sendDest *net.IPAddr
const groupIPProto = "ip:253"
func send(s string) {
	if nSent > maxSends {
		log.Printf("sends capped at %d; not sending %s", maxSends, s)
		return
	}
	atomic.AddInt32(&nSent, 1)
	log.Printf("%20s to %s: %s", "SEND", sendDest.String(), s)
	conn, err := net.DialIP(groupIPProto, nil, sendDest)
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()
	_, err = conn.Write([]byte(s))
	if err != nil {
		log.Panic(err)
	}
}

// This is the recovery log used for persistence of promises and
// accepts.
func logfile(id int) (io.Reader, *log.Logger) {
	s := fmt.Sprintf("upaxos-%d.log", id)
	f, err := os.OpenFile(s, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Panic(err)
	}
	return f, log.New(f, fmt.Sprintf("%d: ", id), 0)
}

func init() {
	flag.IntVar(&myID, "i", -1,
		"identifier for this Paxos participant")
	flag.IntVar(&nGroup, "n", -1,
		"number of Paxos participants")
}
func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	bcastIP := "127.0.0.1"
	flag.Parse()
	if myID == -1 || nGroup == -1 {
		log.Panic("usage")
	}	
	log.Printf("upaxos id(%d) started in group of %d", myID, nGroup)
	defer log.Printf("upaxos id(%d) ending", myID)

	_, lf := logfile(myID)
	lf.Printf("starting %d", myID)
	// XXX TODO: Read in Promises and Accepts from log,
	//	store in data structures,
	//	provide data structures to accept and learn functions
	//	as "priming."

	// begin listening on my well known address
	la, err := net.ResolveIPAddr("ip4", bcastIP)
	if err != nil {
		log.Panic(err)
	}
	conn, err := net.ListenIP(groupIPProto, la)
	if err != nil {
		log.Panic(err)
	}

	sendDest, err = net.ResolveIPAddr("ip4", bcastIP)
	if err != nil {
		log.Panic(err)
	}

	leadc := make(chan Msg)
	acceptc := make(chan Msg)
	learnc := make(chan Msg)
	mainc := make(chan Msg)
	receivers = []chan Msg{leadc, acceptc, learnc, mainc}
	go lead(leadc)
	go accept(acceptc, lf)
	go learn(learnc)
	go listen(conn)
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

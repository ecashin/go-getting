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
//   	       sends Propose, Fix, Fixed
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
//
// Paxos over TCP would be complicated by the extra functionality
// of TCP that insulates software from the realities that Paxos is
// designed for, like failing networks and nodes.  Ideally, you'd
// use the broadcast MAC address with a protocol that's like arp
// or AoE, so that ports and such don't get in the way.
//
// For convenience (non-root development and testing on multiple 
// platforms) UDP is used, but the sender is included in the message.
// RFC 768 says the platform doesn't have to set the port when it is
// not specified by the sender, but you can't listen on one port and
// also use that same port as the source address to send.

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
var receivers []chan Msg

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
}

func respond(conn *net.UDPConn, s string) {
	if conn != nil {
		conn.Write([]byte(s))
	} else {
		// respond to myself
		for _, recvr := range receivers {
			recvr <- Msg{strings.Fields(s), nil}
		}
	}
}

func mustStrtoll(s string) (n int64) {
	n, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		panic(err)
	}
	return
}

// unlike Wikipedia, it's instance first
func instanceProposal(f []string) (i, p int64) {
	i = mustStrtoll(f[1])
	p = mustStrtoll(f[2])
	return
}

// Request message format:
// I	consensus instance (zero for new state)
// V	the value the client wants to set (ignored for lookup)
type Req struct {
	i int64	// 0 for new instance
	v string // ignored for history query
	conn *net.UDPConn
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
	return Req{i, s, m.conn}
}

// Propose message format:
// I	instance
// P	the proposal number the leader is attempting to use
type Propose struct {
	i, p int64
}
func newPropose(f []string) Propose {
	if len(f) < 3 || f[0] != "Propose" {
		log.Panic("called newPropose with bad string")
	}
	i, p := instanceProposal(f)
	return Propose{i, p}
}

// Promise message format is "Promise I A [B V]", where...
// I	instance
// A	the minimum proposal number sender will accept
// B	the proposal number associated with previously accepted value
// V	the previously accepted value
type Promise struct {
	// required fields
	i, p int64

	// optional fields, absent if no prior accepted value
	vp int64	// proposal number associated with the accepted value
	v *string	// the previously accepted value, nil if none accepted
}
func newPromise(f []string) Promise {
	if len(f) < 3 || f[0] != "Promise" {
		log.Panic("called newPromise with bad string")
	}
	i, p := instanceProposal(f)
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
	return Promise{i, p, vp, v}
}

// Accept message format:
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
	i, p := instanceProposal(f)	
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

// Fix message format:
// I	consensus instance
// P	proposal number
// V	value
type Fix struct {
	i, p int64
	v string
}
func newFix(f []string) Fix {
	if len(f) < 3 || f[0] != "Fix" {
		panic("called newAccept with bad string")
	}
	i, p := instanceProposal(f)	
	return Fix{i, p, strings.Join(f[3:], " ")}
}

const maxReqQ = 10	// max 10 queued requests

func lead(c chan Msg, g []string) {
	instance := int64(0)	// consensus instance leader is trying to use
	lastp := int64(myID)	// proposal number last sent
	rq := list.New()	// queued requests
	nrq := 0		// number of queued requests
	var r *Req		// client request in progress
	var v *string	// value to fix
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
		n := int64(len(g))
		p /= n
		p++
		lastp = p * n + int64(myID)
	}
	nextInstance := func() {
		catchup(instance+1, 0)
	}

	everybody := make([]string, len(g)+1)
	for i, _ := range g {
		everybody[i] = g[i]
	}
	everybody[len(g)] = myAddr

	sendall := func(s string) {
		for _, ra := range g {
			if ra != myAddr {
				send(s, ra)
			}
		}
		// send the message to myself, too, all receiver goroutines
		for _, recvr := range receivers {
			recvr <- Msg{strings.Fields(s), nil}
		}
	}
	for {
		select {
		case m := <- c:
			switch m.f[0] {
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
				} else if p.p != lastp {
					catchup(p.i, p.p)
					log.Printf("proposal mismatch (%d %d) try again", p.p, lastp)
					continue
				}
				if p.v != nil {
					if p.vp > vp {
						v = p.v
						vp = p.vp
					}
				}
				npromise++
				log.Printf("got promise from %s\n",
					m.conn.RemoteAddr().String())
				if npromise > len(g)/2 {
					if v == nil {
						v = &r.v
					}
					log.Print("send Fix message", v)
					s := fmt.Sprintf("Fix %d %d %s",
						instance, lastp, v)
					sendall(s)
				}
			case "Accept":
				if r == nil {
					log.Print("ignoring Accept with no Req in progress")
					continue
				}
				a := newAccept(m.f)
				if v == nil || a.v != *v || a.i != instance || a.p != lastp {
					log.Panic("mismatch accept")
				}
				log.Printf("got accept %d %d %v", a.i, a.p, a.v)
				naccepts++
				if naccepts > len(g)/2 {
					if a.v == r.v {
						respond(r.conn, "OK")
						r = nil
						if rq.Front() != nil {
							e := rq.Front()
							r = e.Value.(*Req)
							rq.Remove(e)
						}
					}
					nextInstance()
				}
			case "Request":
				newr := newReq(m)
				if r == nil {
					r = &newr
					s := fmt.Sprintf("Propose %d %d %s",
						instance, lastp, r.v)
					sendall(s)
				} else if nrq < maxReqQ {
					rq.PushBack(newr)
					nrq++
				} else {
					log.Print("send BUSY to client")
				}
			case "NACK":
				nk := newNack(m.f)
				log.Printf("NACK for instance: %d", nk.i)
				catchup(nk.i, nk.p)
			}
		case <- time.After(30 * time.Second):
			log.Print("tick tock")	// demo
		}
	}
}
func accept(c chan Msg) {
	// per-instance record of minimum proposal number we can accept
	minp := make(map[int64]int64)
	accepted := make(map[int64]string)	// values by instance
	for m := range c {
		switch m.f[0] {
		case "Propose":
			p := newPropose(m.f)
			min, present := minp[p.i]
			s := ""
			if !present {
				minp[p.i] = p.p
				s = fmt.Sprintf("Promise %d %d", p.i, p.p)
			} else if p.p < min {
				s = fmt.Sprintf("NACK %d %d", p.i, minp)
			} else {
				s = fmt.Sprintf("Promise %d %d", p.i, p.p)
				if va, there := accepted[p.i]; there {
					s += " " + va
				}
			}
			respond(m.conn, s)
		case "Fix":
			log.Print("received fix")
			fx := newFix(m.f)
			min, there := minp[fx.i]
			s := ""
			if there && min > fx.p {
				log.Printf("acceptor with min %d ignoring Fix %d %d %v",
					min, fx.i, fx.p, fx.v)
				s = fmt.Sprintf("NACK %d %d", fx.i, min)
			} else {
				s = fmt.Sprintf("Accept %d %d", fx.i, fx.p)
				if va, there := accepted[fx.i]; there {
					s += " " + va
				} else {
					accepted[fx.i] = fx.v
					s += " " + fx.v
				}
			}
			respond(m.conn, s)
		}
	}
}

// The Accepts for a given consensus instance
// Storing the proposal number protects against out-of-order delivery
// of accept messages by the network.
type Accepts struct {
	v map[string]string	// accepted values by remote addr
	n map[string]int	// count of hosts by value accepted
	p map[string]int64	// proposal number associated with value accepted by given host
}
func newAccepts() Accepts {
	return Accepts {
		make(map[string]string),
		make(map[string]int),
		make(map[string]int64),
	}
}
func learn(c chan Msg, g []string) {
	history := make(map[int64]Accepts)
	fixed := make(map[int64]string)	// quorum-accepted value by instance
	for m := range c {
		switch m.f[0] {
		case "Accept":
			a := newAccept(m.f)
			if _, ok := fixed[a.i]; ok {
				log.Printf("ignoring Accept for fixed instance %d",
					a.i)
				continue
			}
			if _, ok := history[a.i]; !ok {
				history[a.i] = newAccepts()
			}
			as := history[a.i]
			h := m.conn.RemoteAddr().String()
			oldv, wasThere := as.v[h]
			if wasThere && a.p < as.p[h] {
				continue	// ignore old Accept
			}
			as.v[h] = a.v
			as.p[h] = a.p
			if wasThere {
				as.n[oldv] -= 1
			}
			as.n[a.v] += 1
			log.Printf("learner got \"%s\" from %s, for %d accepts",
				a.v, m.conn.RemoteAddr().String(), as.n[a.v])
			if as.n[a.v] > len(g)/2 {
				fixed[a.i] = a.v
			}
		case "Request":
			r := newReq(m)
			if as, present := history[r.i]; present {
				for v, n := range as.n {
					if n > len(g)/2 {
						s := fmt.Sprintf("Fixed %i %s",
							r.i, v)
						respond(m.conn, s)
						break
					}
				}
			}
		}
	}
}

func listen(conn *net.UDPConn) {
	buf := make([]byte, 9999)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Panic(err)
		}
		s := string(buf[:n])
		f := strings.Fields(s)
		if len(f) == 0 {
			log.Print("skipping zero-field message")
			continue
		}
		for _, c := range receivers {
			c <- Msg{f, conn}
		}
	}
}

type sendMsg struct {
	s, ra string
}
var sendChan chan sendMsg
func send(s, ra string) {
	sendChan <- sendMsg{s, ra}
}
func sender() {
	conns := make(map[string] *net.UDPConn)
	for sm := range sendChan {
		s := sm.s
		ra := sm.ra
		log.Printf("sending to %s: %s", ra, s)
		conn, ok := conns[ra]
		if !ok {
			log.Print("making new connection")
			raddr, err := net.ResolveUDPAddr("udp4", ra)
			if err != nil {
				log.Panic(err)
			}
			conn, err = net.DialUDP("udp4", nil, raddr)
			if err != nil {
				log.Panic(err)
			}
			go listen(conn)
			conns[ra] = conn
		} else {
			log.Print("using existing connection")
		}
		n, err := conn.Write([]byte(s))
		if err != nil {
			log.Panic(err)
		}
		log.Printf("sent %d bytes", n)
	}
}

func init() {
	flag.StringVar(&myAddr, "a", "127.0.0.1:9876",
		"IP and port for this process")
}
func main() {
	flag.Parse()
	g := group()

	log.Print("upaxos started at ", myAddr)
	defer log.Print("upaxos ending")

	// begin listening on my well known address
	la, err := net.ResolveUDPAddr("udp4", myAddr)
	if err != nil {
		log.Panic(err)
	}
	conn, err := net.ListenUDP("udp4", la)
	if err != nil {
		log.Panic(err)
	}

	sendChan = make(chan sendMsg)
	go sender()

	leadc := make(chan Msg)
	acceptc := make(chan Msg)
	learnc := make(chan Msg)
	mainc := make(chan Msg)
	receivers = []chan Msg{leadc, acceptc, learnc, mainc}
	go lead(leadc, g)
	go accept(acceptc)
	go learn(learnc, g)
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

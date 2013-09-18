// upaxos.go - unreliable-broadcast-based Paxos participant

package main

import (
	"bufio"
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

const maxSends = 50 // in case things get out of hand, stop

var nSent int32
var myID int = -1
var nGroup int = -1
var receivers []chan Msg

type Msg struct {
	f []string // whitespace-separated message fields
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
	i int64  // 0 for new instance
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
	vp int64   // proposal number associated with the accepted value
	v  *string // the previously accepted value, nil if none accepted
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
	v       string
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
	v       string
}

func newWrite(f []string) Write {
	if len(f) < 4 || f[1] != "Write" {
		panic("called newWrite with bad string")
	}
	s, i, p := sipParse(f)
	return Write{s, i, p, strings.Join(f[4:], " ")}
}

const maxReqQ = 10 // max 10 queued requests

func lead(c chan Msg) {
	instance := int64(1) // consensus instance leader is trying to use
	lastp := int64(myID) // proposal number last sent
	rq := list.New()     // queued requests
	nrq := 0             // number of queued requests
	var r *Req           // client request in progress
	var v *string        // value to write
	vp := int64(-1)      // proposal number associated with v
	npromise := 0        // number of promises received for r
	naccepts := 0        // number of accepts received for r
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
		lastp = p*n + int64(myID)
	}
	nextInstance := func() {
		catchup(instance+1, 0)
	}
	propose := func() {
		s := fmt.Sprintf("%d Propose %d %d %s",
			myID, instance, lastp, r.v)
		go send(s)
	}

	for {
		select {
		case m := <-c:
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
					propose()
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
					continue // ignore lower-numbered proposals
				} else if p.p > lastp {
					catchup(p.i, p.p) // snoop: like a NACK
					continue
				}
				if p.v != nil {
					if p.vp > vp {
						v = p.v
						vp = p.vp
					}
				}
				npromise++
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
					if r != nil {
						propose()
					}
				}
			case "NACK":
				nk := newNack(m.f)
				if nk.i > instance || nk.p > lastp {
					catchup(nk.i, nk.p)
					if r != nil {
						propose()
					}
				}
			}
		case <-time.After(30 * time.Second):
			log.Print("tick tock") // XXXdemo
		}
	}
}

type Accepted struct {
	p int64
	v string
}

func accept(c chan Msg, lf *log.Logger, lp []loggedPromise, la []loggedAccept) {
	// per-instance record of minimum proposal number we can accept
	minp := make(map[int64]int64)
	accepted := make(map[int64]Accepted) // values by instance

	// first, recover info from logged operations
	for _, rec := range lp {
		minp[rec.i] = rec.p
	}
	for _, rec := range la {
		accepted[rec.i] = Accepted{rec.p, rec.v}
	}

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
				lf.Printf("promise %d %d", p.i, p.p)
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
				lf.Printf("accept %d %d %s", wr.i, wr.p, wr.v)
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
	v map[int64]string // accepted values by participant (host) ID
	n map[string]int   // count of hosts by value accepted
	p map[int64]int64  // proposal number associated with value accepted by given host
}

func newAccepts() Accepts {
	return Accepts{
		make(map[int64]string),
		make(map[string]int),
		make(map[int64]int64),
	}
}
func learn(c chan Msg, lf *log.Logger, ll []loggedLearn) {
	history := make(map[int64]Accepts)
	written := make(map[int64]string) // quorum-accepted value by instance

	// prime written with info recovered from log
	for _, rec := range ll {
		log.Printf("load learned: i:%d v:%s", rec.i, rec.v)
		written[rec.i] = rec.v
	}

	for m := range c {
		if len(m.f) < 2 {
			continue
		}
		if m.f[0] == "Request" {
			r := newReq(m)
			if v, present := written[r.i]; present {
				s := fmt.Sprintf("%d OK %d %s",
					myID, r.i, v)
				go send(s)
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
				continue // ignore old Accept
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
				lf.Printf("learn %d %s", a.i, a.v)
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

type loggedPromise struct {
	i, p int64
}
type loggedAccept struct {
	i, p int64
	v    string
}
type loggedLearn struct {
	i int64
	v string
}

func loadLogData(lf io.Reader) (p []loggedPromise, a []loggedAccept, lrn []loggedLearn) {
	p = []loggedPromise{}
	a = []loggedAccept{}
	lrn = []loggedLearn{}
	r := bufio.NewReader(lf)
	ln, err := r.ReadString('\n')
	for err == nil {
		f := strings.Fields(ln)
		f = f[1:] // ignore myID prefix
		switch f[0] {
		case "promise":
			p = append(p, loggedPromise{
				mustStrtoll(f[1]),
				mustStrtoll(f[2]),
			})
		case "accept":
			a = append(a, loggedAccept{
				mustStrtoll(f[1]),
				mustStrtoll(f[2]),
				f[3],
			})
		case "learn":
			lrn = append(lrn, loggedLearn{
				mustStrtoll(f[1]),
				f[2],
			})
		}
		ln, err = r.ReadString('\n')
	}
	return
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

	lfr, lfw := logfile(myID)
	promises, accepts, learnings := loadLogData(lfr)
	lfw.Printf("starting %d", myID)

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
	go accept(acceptc, lfw, promises, accepts)
	go learn(learnc, lfw, learnings)
	go listen(conn)
loop:
	for m := range mainc {
		if len(m.f) > 0 {
			switch m.f[0] {
			case "quit":
				fallthrough
			case "exit":
				fallthrough
			case "bye":
				log.Print("exiting")
				break loop
			}
		}
	}
}

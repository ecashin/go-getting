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
// The main thread handles startup and message multiplexing for
// cooperating goroutines:
//
//   listener: receives messages
//
//   leader:   handles to-leader messages
//   	       sends from-leader messages
//
//   acceptor: handles to-acceptor messages
//   	       sends from-acceptor messages
//
// Each of the latter two has its own state machine.

package main

import (
	"bufio"
	"container/list"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
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

func serve(c chan<- Msg) {
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
		c <- Msg{s, conn, raddr}
	}
}

func send(s string, conn *net.UDPConn, raddr *net.UDPAddr) {
	_, err := conn.WriteToUDP([]byte(s), raddr)
	if err != nil {
		log.Panic(err)
	}
}

func lead(mc <-chan Msg) {
	for {
		m := <- mc
		log.Print("leader sees ", m.s)
		send("OK from leader\n", m.conn, m.raddr)
	}
}

func accept(mc <-chan Msg) {
	for {
		m := <- mc
		log.Print("acceptor sees ", m.s)
		send("OK from acceptor\n", m.conn, m.raddr)
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
	sc := make(chan Msg)
	lc := make(chan Msg)
	ac := make(chan Msg)
	go serve(sc)
	go lead(lc)
	go accept(lc)
	i := 0
	for {
		select {
		case m := <- sc:
			log.Print(m.s, m.raddr)
			flds := strings.Fields(m.s)
			if len(flds) != 0 {
				switch flds[0] {
				case "request": fallthrough
				case "accept": fallthrough
				case "promise":
					lc <- m
				case "propose": fallthrough
				case "set":
					ac <- m
				}
			}
		case <- time.After(1000 * time.Second):
			log.Print("timeout at iteration ", i)
		}
		i++
	}
}

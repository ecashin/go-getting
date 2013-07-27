// Practice responding to messages.
// 
// 
// Start this one first:
// 
// paxos$ sudo go run ~/git/go-getting/paxos/iptest-rsp.go 
// 2013/07/26 21:35:03 95717 starting
// 2013/07/26 21:35:09 received from 127.0.0.1: Request 0 hi
// 2013/07/26 21:35:09 sending to 127.0.0.1: 95717(Request 0 hi)
// 2013/07/26 21:35:09 sent 19 bytes
// 2013/07/26 21:35:10 received from 127.0.0.1: 95717(Request 0 hi)
// 2013/07/26 21:35:10 sending to 127.0.0.1: 95717(95717(Request 0 hi))
// 2013/07/26 21:35:10 sent 26 bytes
// 2013/07/26 21:35:10 received from 127.0.0.1: 95722(Request 0 hi)
// 2013/07/26 21:35:10 sending to 127.0.0.1: 95717(95722(Request 0 hi))
// 2013/07/26 21:35:10 sent 26 bytes
// paxos$ 
// 
// Next start another one:
// 
// ~$ sudo go run ~/git/go-getting/paxos/iptest-rsp.go 
// 2013/07/26 21:35:06 95722 starting
// 2013/07/26 21:35:09 received from 127.0.0.1: Request 0 hi
// 2013/07/26 21:35:09 sending to 127.0.0.1: 95722(Request 0 hi)
// 2013/07/26 21:35:09 sent 19 bytes
// 2013/07/26 21:35:09 received from 127.0.0.1: 95717(Request 0 hi)
// 2013/07/26 21:35:09 sending to 127.0.0.1: 95722(95717(Request 0 hi))
// 2013/07/26 21:35:09 sent 26 bytes
// 2013/07/26 21:35:09 received from 127.0.0.1: 95722(Request 0 hi)
// 2013/07/26 21:35:09 sending to 127.0.0.1: 95722(95722(Request 0 hi))
// 2013/07/26 21:35:10 sent 26 bytes
// ~$ 
// 
// Then give them something to talk about:
// 
// ~$ echo Request 0 hi | sudo go run ~/git/go-getting/paxos/iptest-send.go -a 127.0.0.1 -p 253
// ~$ 

package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)
const groupIPProto = "ip:253"
var addr *net.IPAddr
var pid int

func send(s string) {
	log.Printf("sending to %s: %s", addr.String(), s)
	conn, err := net.DialIP(groupIPProto, nil, addr)
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()
	n, err := conn.Write([]byte(s))
	if err != nil {
		log.Panic(err)
	}
	log.Printf("sent %d bytes", n)
}

func main() {
	pid = os.Getpid()
	log.Printf("%d starting", pid)
	a, err := net.ResolveIPAddr("ip4", "127.0.0.1")
	if err != nil {
		log.Panic(err)
	}
	addr = a
	conn, err := net.ListenIP("ip:253", addr)
	if err != nil {
		log.Panic(err)
	}
	buf := make([]byte, 9999)
	i := 0
	lim := 3
	for {
		n, ra, err := conn.ReadFromIP(buf)
		if err != nil {
			log.Panic(err)
		}
		s := string(buf[:n])
		s = strings.TrimSpace(s)
		log.Printf("received from %s: %s", ra.String(), s)
		if i < lim {
			i++
			go send(fmt.Sprintf("%d:%d(%s)", pid, i, s))
		}
	}
}

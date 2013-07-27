// Practice responding to messages.
// 
// ~$ sudo go run ~/git/go-getting/paxos/iptest-rsp.go 
// 2013/07/26 21:21:25 received from 127.0.0.1: Request 0 hi
// 2013/07/26 21:21:25 sending to 127.0.0.1: pluple Request 0 hi
// 2013/07/26 21:21:25 sent 20 bytes
// 2013/07/26 21:21:25 received from 127.0.0.1: pluple Request 0 hi
// 2013/07/26 21:21:25 sending to 127.0.0.1: pluple pluple Request 0 hi
// 2013/07/26 21:21:25 sent 27 bytes
// 2013/07/26 21:21:25 received from 127.0.0.1: pluple pluple Request 0 hi
// 2013/07/26 21:21:25 sending to 127.0.0.1: pluple pluple pluple Request 0 hi
// 2013/07/26 21:21:25 sent 34 bytes
// ~$ 
// 
// Meanwhile,
// 
// ~$ echo Request 0 hi | sudo go run ~/git/go-getting/paxos/iptest-send.go -a 127.0.0.1 -p 253
// ~$ 

package main

import (
	"log"
	"net"
)
const groupIPProto = "ip:253"
var addr *net.IPAddr
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
	i := 3
	for i > 0 {
		i--
		n, ra, err := conn.ReadFromIP(buf)
		if err != nil {
			log.Panic(err)
		}
		s := string(buf[:n])
		log.Printf("received from %s: %s", ra.String(), s)
		send("pluple " + s)
	}
}

// Let's try to replicate the pattern of concurrent listening and
// sending here.

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"
)

const basePort = 4568

var amTwo bool

func listen(c chan string, conn *net.UDPConn, label string) {
	log.Printf("listening to %s", conn.LocalAddr().String())
	buf := make([]byte, 9999)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Print(err)
			time.Sleep(3 * time.Second)
			continue
		}
		s := string(buf[:n])
		log.Printf("%s got {%s}", label, s)
		c <- s
	}
}
func listenPub(c chan string, laddr string) {
	la, err := net.ResolveUDPAddr("udp4", laddr)
	if err != nil {
		log.Panic(err)
	}
	conn, err := net.ListenUDP("udp4", la)
	if err != nil {
		log.Panic(err)
	}
	listen(c, conn, laddr)
}

func init() {
	flag.BoolVar(&amTwo, "t", false,
		"am I the \"second\" guy?")
}
func main() {
	lPubPort := basePort
	rPubPort := basePort
	if amTwo {
		rPubPort++
	} else {
		lPubPort++
	}
	incoming := make(chan string)
	go listenPub(incoming, fmt.Sprintf("127.0.0.1:%d", lPubPort))
	time.Sleep(3 * time.Second)
	ra := fmt.Sprintf("127.0.0.1:%d", rPubPort)
	var conn *net.UDPConn
	for {
		log.Printf("lPubPort:%d sending to %s", lPubPort, ra)
		raddr, err := net.ResolveUDPAddr("udp4", ra)
		if err != nil {
			log.Print(err)
			time.Sleep(3 * time.Second)
			continue
		}
		conn, err = net.DialUDP("udp4", nil, raddr)
		if err != nil {
			log.Print(err)
			time.Sleep(3 * time.Second)
			continue
		}
		msg := fmt.Sprintf("Hello from %d", lPubPort)
		_, err = conn.Write([]byte(msg))
		if err != nil {
			log.Print(err)
			time.Sleep(3 * time.Second)
			continue
		}
		break
	}
	go listen(incoming, conn, "response")
	i := 5
	for m := range incoming {
		i--
		if m == "quit" || i < 0 {
			break
		}
	}
}

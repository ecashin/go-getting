// node.go - two-phase commit demo
// There's the coordinator and the drone.

package main

import (
//	"bytes"
	"flag"
	"fmt"
//	"io"
	"log"
	"net"
	"os"
	"strings"
)

func serve(c chan string) {
	la, err := net.ResolveUDPAddr("udp4", laddr)
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
		log.Printf("%s says %s", raddr, s)
		c <- s
		rsp := <- c
		log.Printf("responding to %s with %s", raddr, rsp)
		_, err = conn.WriteToUDP([]byte(rsp), raddr)
		if err != nil {
			log.Panic(err)
		}
	}
	c <- "EOF"
}

func startLog() *log.Logger {
	logd := fmt.Sprintf("%s/tmp/node.go", os.Getenv("HOME"))
	logf := fmt.Sprintf("log-%d", os.Getpid())
	if err := os.MkdirAll(logd, 0755); err != nil {
		log.Panic(err)
	}
	outlog, err := os.OpenFile(fmt.Sprintf("%s/%s", logd, logf),
		os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	return log.New(outlog, "", log.LstdFlags|log.Lmicroseconds)
}

var laddr string
var coordinate bool	// XXXunfinished: not enforced
var value = "(unset value)"

func init() {
	flag.StringVar(&laddr, "p", "127.0.0.1:9999",
		"the laddr to talk to")
	flag.BoolVar(&coordinate, "c", false,
		"whether to be the coordinator")
}
func main() {
	flag.Parse()

	// this is the two-phase commit log on stable storage
	l := startLog()
	l.Print("starting log")

	c := make(chan string)
	go serve(c)
	log.Print("started server")

	state := "listening"
	req := "(no request)"

	// the coordinator gets different messages than the drone
	for {
		s := <-c
		f := strings.Fields(s)
		switch strings.ToLower(f[0]) {
		default:
			c <- (f[0] + " not good for me\n")
		// messages sent to coordinator:
		case "request":
			switch state {
			case "listening":
				req = strings.Join(f[1:], " ")
				msg := fmt.Sprintf("prepare %s", req)
				l.Print(msg)
				state = "prep"
				c <- (msg + "\n")
			default:
				log.Panic("wasn't listening")
			}
		case "yes":
			switch state {
			case "prep":
				msg := fmt.Sprintf("commit %s", req)
				l.Print(msg)
				value = req
				state = "listening"
				c <- (msg + "\n")
			default:
				log.Panic("wasn't preparing")
			}
		case "no":
			switch state {
			case "prep":
				msg := fmt.Sprintf("abort %s", req)
				l.Print(msg)
				state = "listening"
				// old value unaffected by transaction
				c <- (msg + "\n")
			default:
				log.Panic("wasn't preparing")
			}
			log.Print("drone talking")
		// messages sent from coordinator:
		case "prepare":
			log.Fatal("coordinator talking")
		case "commit":
			log.Fatal("coordinator talking")
		case "abort":
			log.Fatal("coordinator talking")
		// messages that are not part of 2PC but are handy
		case "peek":
			c <- (value + "\n")
		case "quit":
			log.Fatal("quitting by remote request")
		}
	}
}

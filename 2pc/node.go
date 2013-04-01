// node.go - two-phase commit demo
// There's the coordinator and the cohort.
// This is a presume-abort variant of the 2PC. (See Lampson and Lomet 1993)
//
// The coordinator listens for requests from clients, and it
// dials the (sole, for now) cohort.  The cohort listens for
// messages from the coordinator.

package main

import (
//	"bytes"
	"flag"
	"fmt"
//	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"
)

func serve(c chan string, myAddr string) {
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
		log.Printf("serve: %s says %s; sending to state machine", raddr, s)
		c <- s
		rsp := <- c
		log.Printf("serve: responding to %s with %s", raddr, rsp)
		_, err = conn.WriteToUDP([]byte(rsp), raddr)
		if err != nil {
			log.Panic(err)
		}
	}
}

func dial(client, stateMach chan string, theirAddr string) {
	conn, err := net.Dial("udp", theirAddr)
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()
	buf := make([]byte, 9999)
	for {
		var msg string
		select {
		case msg = <- stateMach:
		case msg = <- client:
		}
		log.Printf("dial: sending \"%s\" to %s", msg, theirAddr)
		_, err := conn.(*net.UDPConn).Write([]byte(msg))
		if err != nil {
			log.Panic(err)
		}
		n, raddr, err := conn.(*net.UDPConn).ReadFromUDP(buf)
		if err != nil {
			log.Panic(err)
		}
		s := string(buf[:n])
		log.Printf("dial: %s says %s; sending to state machine", raddr, s)
		stateMach <- s
	}
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

func pause() {
	time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
}

const coordAddr = "127.0.0.1:9898"
const cohortAddr = "127.0.0.1:9999"
var doCoordinate bool
var value = "(unset value)"

func init() {
	flag.BoolVar(&doCoordinate, "c", false,
		"whether to be the coordinator")
}
func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	// this is the two-phase commit log on stable storage
	l := startLog()
	l.Print("starting log")

	srvc := make(chan string)
	dialc := make(chan string)
	if doCoordinate {
		go serve(srvc, coordAddr)
		log.Print("started server on ", coordAddr)
		go dial(srvc, dialc, cohortAddr)
		log.Print("started dialer to ", cohortAddr)
	} else {
		go serve(srvc, cohortAddr)
		log.Print("started server on ", cohortAddr)
	}

	state := "listening"
	req := "(no request)"

	// the coordinator gets different messages than the cohort
	for {
		var s string
		var cp *chan string
		select {
		case s = <-srvc:
			cp = &srvc
		case s = <-dialc:
			cp = &dialc
		}
		f := strings.Fields(s)
		switch strings.ToLower(f[0]) {
		default:
			*cp <- (f[0] + " not good for me\n")
		// messages sent to coordinator:
		case "request":
			switch state {
			case "listening":
				req = strings.Join(f[1:], " ")
				msg := fmt.Sprintf("prepare %s", req)
				l.Print(msg)
				state = "prep"
				dialc <- msg
			default:
				log.Panic("wasn't listening")
			}
		case "yes":
			switch state {
			case "prep":
				final := "commit"
				if rand.Intn(10) > 6 {
					final = "abort"
				}
				msg := fmt.Sprintf("%s %s", final, req)
				l.Print(msg)
				if final == "commit" {
					value = req
				}
				state = "listening"
				pause()
				*cp <- msg
				if final == "commit" {
					srvc <- ("OK" + "\n")
				} else {
					srvc <- ("SORRY" + "\n")
				}
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
				*cp <- msg
			default:
				log.Panic("wasn't preparing")
			}
		case "ack":
			switch state {
			case "listening":
				// listen for more requests from clients
			default:
				log.Panic("wasn't listening")
			}
		// messages sent from coordinator:
		case "prepare":
			switch state {
			case "listening":
				agree := "yes"
				if rand.Intn(10) > 6 {
					agree = "no"
				}
				msg := agree
				l.Print(msg)
				if agree == "yes" {
					state = "uncertain"
				} else {
					state = "listening"
				}
				pause()
				*cp <- msg
			default:
				log.Fatal("cohort wasn't listening")
			}
		case "commit":
			switch state {
			case "uncertain":
				l.Print("commit " + req)
				value = req
				state = "listening"
				*cp <- "ack"
			default:
				log.Fatal("cohort wasn't uncertain")
			}
		case "abort":
			switch state {
			case "uncertain":
				l.Print("abort " + req)
				state = "listening"
				*cp <- "ack"
			case "listening":
				l.Print("abort " + req)
				*cp <- "ack"
			default:
				log.Fatal("cohort wasn't listening")
			}
		// messages that are not part of 2PC but are handy
		case "peek":
			*cp <- (value + "\n")
		case "quit":
			log.Fatal("quitting by remote request")
		}
	}
}

// node.go - two-phase commit demo
// There's the coordinator and the cohort.
// This is a presume-abort variant of the 2PC. (Lampson and Lomet 1993)
//
// The coordinator listens for requests from clients, and it
// dials the (sole, for now) cohort.  The cohort listens for
// messages from the coordinator.
//
// Example usage with three processes on term1, term2, term3:
// term1$ go run node.go -c	# run the coordinator
// term2$ go run node.go	# run the cohort
// term3$ nc -u localhost 9898	# interact with coordinator
//
// By default, there will be some simulated drops of packets.
// You can use the "-d" option to specify a ratio of drops to
// total packets.

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
		if len(strings.Fields(s)) == 0 {
			continue
		}
		log.Printf("serve: %s says %s; sending to state machine", raddr, s)
		c <- s
		rsp := <- c
		log.Printf("serve: responding to %s with %s", raddr, rsp)
		if (!drop()) {
			_, err = conn.WriteToUDP([]byte(rsp), raddr)
			if err != nil {
				log.Panic(err)
			}
		}
	}
}

func drop() bool {
	d := rand.Float64() <= dropRatio
	if (d) {
		log.Print("packet DROP!")
	}
	return d
}

func dial(stateMach chan string, theirAddr string) {
	conn, err := net.Dial("udp", theirAddr)
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()
	buf := make([]byte, 9999)
	udp := make(chan string)
	for {
		msg := <- stateMach
		log.Printf("dial: sending \"%s\" to %s", msg, theirAddr)
		if (!drop()) {
			_, err := conn.(*net.UDPConn).Write([]byte(msg))
			if err != nil {
				log.Panic(err)
			}
		}
		var raddr *net.UDPAddr
		var err error
		go func() {
			var n int
			n, raddr, err = conn.(*net.UDPConn).ReadFromUDP(buf)
			if err != nil {
				log.Panic(err)
			}
			udp <- string(buf[:n])
		} ()
		var s string
		select {
		case <- time.After(2*time.Second):
			s = "timeout"
			log.Print("dial: TIMEOUT reading from UDP")
		case s = <- udp:
			log.Printf("dial: %s says %s; sending to state machine", raddr, s)
		}
		stateMach <- s
	}
}

// returns the log, the value, and whether the value is uncertain
func startLog() (*log.Logger, string, bool) {
	logd := fmt.Sprintf("%s/tmp/node.go", os.Getenv("HOME"))
	logf := "cohort.log"
	if doCoordinate {
		logf = "coordinator.log"
	}
	if err := os.MkdirAll(logd, 0755); err != nil {
		log.Panic(err)
	}
	l, err := os.OpenFile(fmt.Sprintf("%s/%s", logd, logf),
		os.O_SYNC|os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	bufsiz := 9000
	buf := make([]byte, bufsiz)
	n, err := l.Seek(0, os.SEEK_END)
	if err != nil {
		log.Panic(err)
	}
	if n > int64(bufsiz) {
		n = int64(bufsiz)
	}
	_, err = l.Seek(-int64(n), os.SEEK_END)
	if err != nil {
		log.Panic(err)
	}
	m := 0
	if n > 0 {
		m, err = l.Read(buf)
		if err != nil {
			log.Panic(err)
		}
	}
	value := "(unset value)"
	uncertain := false
	if m > 0 {
		lines := strings.FieldsFunc(string(buf[:m]), func (c rune) bool {
			return c == '\n'
		})
		for _, i := range lines {
			log.Print(logf+": "+i)
			f := strings.Fields(i)
			if len(f) > 2 {
				v := ""
				if len(f) > 3 {				
					v = strings.Join(f[3:], " ")
				}
				switch f[2] {
				case "commit":
					uncertain = false
					value = v
				case "yes":
					uncertain = true
					value = v
				}
			}
		}
	}
	return log.New(l, "", log.LstdFlags|log.Lmicroseconds), value, uncertain
}

func pause() {
	time.Sleep(time.Duration(rand.Intn(400)) * time.Millisecond)
}

const coordAddr = "127.0.0.1:9898"
const cohortAddr = "127.0.0.1:9999"
var doCoordinate bool
var dropRatio float64

func init() {
	flag.BoolVar(&doCoordinate, "c", false,
		"whether to be the coordinator")
	flag.Float64Var(&dropRatio, "d", 0.05,
		"dropped/total ratio for sent UDP packets")
}
func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	// this is the two-phase commit log on stable storage
	l, value, uncertain := startLog()
	state := "listening"
	prefix := "START"
	if uncertain {
		if doCoordinate {
			log.Fatal("uncertain coordinator")
		}
		prefix += " UNCERTAIN"
		state = "uncertain"
	}
	l.Printf("%s process in state(%s) with value(%s)", prefix, state, value)
	srvc := make(chan string)
	dialc := make(chan string)
	if doCoordinate {
		go serve(srvc, coordAddr)
		log.Print("started server on ", coordAddr)
		go dial(dialc, cohortAddr)
		log.Print("started dialer to ", cohortAddr)
	} else {
		go serve(srvc, cohortAddr)
		log.Print("started server on ", cohortAddr)
	}
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
		case "req":
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
				if rand.Intn(10) > 8 {
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
				srvc <- ("SORRY" + "\n")
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
		// internal messages:
		case "timeout":
			if doCoordinate {
				switch state {
				case "prep":	// same as getting "no"
					msg := fmt.Sprintf("abort %s", req)
					l.Print(msg)
					state = "listening"
					*cp <- msg
					srvc <- ("SORRY" + "\n")
				case "listening":
					// noop
				default:
					log.Panic("unsupported timeout in coordinator")
				}
			} else {
				switch state {
				case "uncertain":
					*cp <- "peek"	// ask what the value is
				default:
					log.Panic("unsupported timeout in cohort")
				}
			}
		// messages sent from coordinator:
		case "value":
			switch state {
			case "uncertain":
				value = strings.Join(f[1:], " ")
				l.Print("commit " + value)
				state = "listening"
				*cp <- "ack"
			default:
				log.Fatal("cohort wasn't uncertain")
			}
		case "prepare":
			switch state {
			case "listening":
				agree := "yes"
				if rand.Intn(10) > 8 {
					agree = "no"
				}
				msg := agree
				req = ""
				if len(f) > 1 {
					req = strings.Join(f[1:], " ")
					msg += " " + req
				}
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
			*cp <- ("value " + value)
		case "quit":
			log.Fatal("quitting by remote request")
		}
	}
}

// test IRC client uses https://github.com/husio/go-irc.git
// based on examples/client.go in go-irc
//
// GOPATH="$HOME"/git/go-irc go run pircxos.go
//
// Message formats:
//
// G P propose		leader (AKA proposer) proposes number P for 
// 			game G
//
// G P promise Q V	acceptor promises not to accept proposal
// 			with number less than P in game G, telling
//			proposer that value V has already been accepted
//			for proposal number Q in game G.  A nil V means
//			no value has been accepted yet.
// 
// G P set V		leader asks acceptors to accept value
//			V for proposal number P in game G.  It should
//			be the V with the highest Q with a non-nil V,
//			or a V of the leaders chosing if no promise had
//			a non-nil V.
//
// G P accept V		acceptor accepts value V.  This acceptor will
//			accept a different value with a higher proposal
//			number, though, if one comes.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"irc"
	"log"
	"os"
	"regexp"
	"strings"
)

var server *string = flag.String("server", "irc.freenode.net", "IRC server address")
var port *int = flag.Int("port", 6667, "IRC server port")
var modnick *string = flag.String("nick", "go-irc-client", "Nickname")

type Promise struct {
	acceptor string
	val      *string
}

type Acceptor struct {
	min       int64   // player promised not to accept proposals less than this
	paccepted int64   // proposal number of accepted value
	aVal      *string // the accepted value itself
}

type Leader struct {
	lVal     string
	promises []Promise
	accepts  []string // players who have accepted proposed value
}

type Player struct {
	id   int   // -1 for "not playing"
	seen int64 // largest proposal number observed so far
	Acceptor
	Leader
}

type PMod struct {
	ircchan string
	gameno  int64 // game instance number
	players map[string]Player
}

// :wonkawonka!~ecashin@blah.example.com PRIVMSG #pmodtesting :hi
func nick(privmsg string) string {
	i := strings.Index(privmsg, ":")
	j := strings.Index(privmsg, "!")
	if i < 0 || j < 0 || j-i < 1 {
		panic("no nick in privmsg")
	}
	return privmsg[i+1 : j]
}

func (pm *PMod) playerstr() string {
	return fmt.Sprintf("%d", len(pm.players))
}

//  :bobobobono!~ecashin@hosty.example.com JOIN #pmodtesting
//  :bobobobono!~ecashin@hosty.example.com QUIT :Client Quit
//  :wowowowowon!~ecashin@hosty.example.com JOIN #pmodtesting
//  :wowowowowon!~ecashin@hosty.example.com PART #pmodtesting
//  PING :verne.freenode.net
//  :wowowowowon!~ecashin@hosty.example.com JOIN #pmodtesting
//  :wowowowowon!~ecashin@hosty.example.com PRIVMSG #pmodtesting :hello there
func (pm *PMod) handle(send func(string), m string) bool {
	cont := true
	re := regexp.MustCompile(":(\\w+?)!~(\\S+?)@(\\S+?)\\s+(\\S+)\\s+(.*)")
	g := re.FindStringSubmatch(m)
	for i, v := range g {
		log.Println(i, v)
	}
	search := " PRIVMSG #" + pm.ircchan + " :"
	i := strings.Index(m, search)
	log.Printf("i:%d", i)
	if i > 0 {
		log.Printf("m[i+1:] \"%s\"", m[i+1:])
	}
	if i > 0 && i+len(search) < len(m) {
		switch m[i+len(search):] {
		case "join":
			nam := nick(m)
			if _, present := pm.players[nam]; !present {
				id := len(pm.players)
				pm.players[nam] = Player{
					id: id,
				}
				pm.gameno++
				send("PRIVMSG #" + pm.ircchan + fmt.Sprintf(" :NEW GAME: %d", pm.gameno))
				send("PRIVMSG #" + pm.ircchan + " :Players: " + pm.playerstr())
			}
		case "go away":
			cont = false
			send("QUIT :going away now")
		}
	}
	return cont
}

func main() {
	flag.Parse()

	addr := fmt.Sprintf("%s:%v", *server, *port)
	c, err := irc.Dial(addr)
	if err != nil {
		fmt.Println(err)
		return
	}
	send := func(s string) {
		fmt.Println("> " + s)
		c.ToSend <- s
	}
	pm := &PMod{
		ircchan: "pmodtesting",
		players: make(map[string]Player),
	}

	quit := make(chan bool)
	send("NICK " + *modnick)
	send("USER ircctest * * :Ed Cashin")
	send("JOIN #" + pm.ircchan)

	// irc messages reader
	go func() {
		for {
			select {
			case err := <-c.Error:
				fmt.Println("client read error", err)
				quit <- true
				return
			case msg := <-c.Received:
				if msg != nil {
					s := msg.String()
					fmt.Println("< ", s)
					if !pm.handle(send, s) {
						return
					}
				} else {
					return
				}
			}
		}
	}()

	// user input reader
	go func() {
		in := bufio.NewReader(os.Stdin)
		for {
			data, err := in.ReadString('\n')
			if err != nil {
				fmt.Sprintf("client write error: %s", err)
				return
			}
			data = strings.TrimSpace(data)
			switch data {
			case "quit":
				send("QUIT :bye")
				quit <- true
			default:
				send(data)
			}
		}
	}()

	<-quit
}

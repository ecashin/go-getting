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
	"strconv"
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
	nick string
	seen int64 // largest proposal number observed so far
	Acceptor
	Leader
}

type PMod struct {
	ircchan string
	gameno  int64 // game instance number
	players map[string]Player
}

func (pm *PMod) playerlines() []string {
	n := len(pm.players)
	a := make([]string, n+1)
	i := 0
	a[i] = fmt.Sprintf("%d players", len(pm.players))
	i++
	for nick, v := range pm.players {
		a[i] = fmt.Sprintf("  id:%d nick:%-15s seen:%d\n", v.id, nick, v.seen)
		i++
	}
	return a
}

func (pm *PMod) handleMsg(send func(string), nick, msg string) {
	ssend := func(s string) {
		send("PRIVMSG #" + pm.ircchan + " :" + s)
	}
	f := strings.Fields(msg)
	if len(f) < 4 {
		ssend("uh ... whatever.")
		return
	}
	ircch, game, proposal, op := f[0], f[1], f[2], f[3]
	if ircch != "#" + pm.ircchan {
		ssend(nick + ": we're talking in #" + pm.ircchan)
		return
	}
	game = game[1:]
	g, err := strconv.ParseInt(game, 0, 64)
	if err != nil || g != pm.gameno {
		ssend(fmt.Sprintf("we're playing game %d, not %s", pm.gameno, game))
		return
	}
	p, err := strconv.ParseInt(proposal, 0, 64)
	if err != nil {
		ssend(fmt.Sprintf("hmm.  \"%s\" doesn't look like a proposal number.", pm.gameno, game))
		return
	}
	switch op {
	default:
		ssend("unknown operation: " + op)
	}
	ssend(fmt.Sprintf("proposal %d", p))
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
	if g == nil {
		return cont
	}
	_, nick, user, host, op, rest := g[0], g[1], g[2], g[3], g[4], g[5]
	log.Println(nick, user, host, op, rest)
	newgame := func() {
		pm.gameno++
		send("PRIVMSG #" + pm.ircchan + fmt.Sprintf(" :NEW GAME: %d", pm.gameno))
		for _, line := range pm.playerlines() {
			send("PRIVMSG #" + pm.ircchan + " : " + line)
		}
	}
	switch op {
	case "JOIN":
		if _, present := pm.players[nick]; !present {
			id := len(pm.players)
			pm.players[nick] = Player{
				id: id,
			}
			newgame()
		}
	case "PART":
		if _, present := pm.players[nick]; present {
			delete(pm.players, nick)
			i := 0
			for k, v := range pm.players {
				v.id = i
				pm.players[k] = v
				i++
			}
			newgame()
		}
	case "PRIVMSG":
		pm.handleMsg(send, nick, rest)
	default:
		log.Println("unknown op:", op)
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
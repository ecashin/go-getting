// test IRC client uses https://github.com/husio/go-irc.git
// based on examples/client.go in go-irc
//
// This was a cute idea: Create an IRC "bot" that knows how
//   Paxos is supposed to work; then allow humans to execute
//   the algorithm in an IRC channel, with the bot acting as
//   moderator.  In the end, cuteness wasn't enough to justify
//   the effort, and I switched to working on upaxos.go.
//
// GOPATH="$HOME"/git/go-irc go run pircxos.go
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

// Message formats:
//
const proposalFormat = `
G P propose
        leader (AKA proposer) proposes number P for
        game G
`
const promiseFormat = `
G P promise Q V
       acceptor promises not to accept proposal
       with number less than P in game G, telling
       proposer that value V has already been accepted
       for proposal number Q in game G.  If Q and
       V are absent, no value has been accepted yet.
`

var server *string = flag.String("server", "irc.freenode.net", "IRC server address")
var port *int = flag.Int("port", 6667, "IRC server port")
var modnick *string = flag.String("nick", "go-irc-client", "Nickname")

type Promise struct {
	acceptor string
	proposal int64
	val      *string
}

type Acceptor struct {
	min       int64   // player promised not to accept proposals less than this
	pAccepted int64   // proposal number of accepted value
	aVal      *string // the accepted value itself
}

type Leader struct {
	lProposed int64      // this Leader did issue this proposal number
	lSet      *string    // this Leader did attempt to set this value
	agenda    string    // value this Leader would like to set
	promises  []Promise // promises received from Acceptors
	accepts   []string  // Players who have accepted proposed value
}

type Player struct {
	id   int // -1 for "not playing"
	nick string
	seen int64 // largest proposal number observed so far
	Acceptor
	Leader
}

type PMod struct {
	ircchan string
	gameno  int64 // game instance number
	players map[string]*Player
}

func (pm *PMod) newProposed(player *Player, p int64) {
	player.lProposed = p
	player.promises = nil
	player.accepts = nil
}

func (pm *PMod) wasProposed(p int64) bool {
	for _, pl := range pm.players {
		if pl.lProposed == p {
			return true
		}
	}
	return false
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

var helplines = []string{
	"this",
	"is",
	"help",
}

func (pm *PMod) maxProposal() int64 {
	prop := int64(-1)
	for _, p := range pm.players {
		if p.seen > prop {
			prop = p.seen
		}
	}
	return prop
}

// returns:
// 1. whether a quorum have promised to accept no lower proposal
// 2. the Promise with the highest-number proposal accepted
// 3. an explanation if 1. is false
func (pm *PMod) quorumPromised(nick string, proposal int64) (bool, *Promise, string) {
	pl, present := pm.players[nick]
	if !present {
		return false, nil, nick + " is not playing"
	}
	if pl.lProposed != proposal {
		return false, nil, fmt.Sprintf("%s proposed %d, not %d", nick, pl.lProposed, proposal)
	}
	maj := len(pm.players) / 2 + 1
	if len(pl.promises) < maj {
		return false, nil, fmt.Sprintf("%s received only %d of %d required promises",
			nick, len(pl.promises), maj)
	}
	var max *Promise
	for _, v := range pl.promises {
		if v.val != nil && (max == nil || v.proposal > max.proposal) {
			max = &v
		}
	}
	return true, max, "ok"
}

func (pm *PMod) wasSet(prop int64, val string) bool {
	pset := false
	valseen := false
	for _, pl := range pm.players {
		if pl.lSet == nil {
			continue
		}
		if pl.lProposed == prop {
			pset = true
		}
		if *pl.lSet == val {
			valseen = true
		}
	}
	return pset && valseen
}

func (pm *PMod) isInvalidProposal(f []string) bool {
	return len(f) > 0
}

// empty value is OK
func (pm *PMod) isInvalidPromise(f []string) bool {
	if len(f) == 0 {
		return false
	}
	fmt.Println("parseint: ", f[0])
	if _, err := strconv.ParseInt(f[0], 0, 64); err != nil {
		return true
	}
	if len(f) > 3 {
		if _, err := strconv.ParseInt(f[3], 0, 64); err != nil {
			return true
		}
	}
	return false
}

func (pm *PMod) handleMsg(send func(string), nick, msg string) {
	csend := func(s string) {
		send("PRIVMSG #" + pm.ircchan + " :" + s)
	}
	csendm := func(s string) {
		for _, i := range strings.Split(s, "\n") {
			if i == "" {
				i = " "
			}
			csend(i)
		}
	}
	psend := func(n, s string) {
		send("PRIVMSG " + n + " :" + s)
	}
	f := strings.Fields(msg)
	if len(f) < 1 {
		csend("confusing stuff...")
		return
	}
	ircch := f[0]
	if ircch != "#"+pm.ircchan {
		csend(nick + ": we're talking in #" + pm.ircchan)
		return
	}
	cmd := f[1]
	cmd = cmd[1:]
	switch cmd { // in case it's not a game message but a special command
	case "status":
		for _, line := range pm.playerlines() {
			psend(nick, line)
		}
		return
	case "help":
		for _, line := range helplines {
			psend(nick, line)
		}
		return
	}
	if len(f) < 4 {
		csend("uh ... whatever.")
		return
	}
	game, proposal, op := cmd, f[2], f[3]
	g, err := strconv.ParseInt(game, 0, 64)
	if err != nil || g != pm.gameno {
		csend(fmt.Sprintf("please ignore %s saying \"%s\".  we're playing game %d",
			nick, game, pm.gameno))
		return
	}
	p, err := strconv.ParseInt(proposal, 0, 64)
	if err != nil {
		csend(fmt.Sprintf("hmm.  \"%s\" doesn't look like a proposal number.",
			proposal))
		csend(fmt.Sprintf("the max proposal number used for game %d was %d.",
			pm.gameno, pm.maxProposal()))
		return
	}
	talker, present := pm.players[nick]
	if !present {
		csend(nick + ": you're not playing.  try re-joining " + pm.ircchan)
		return
	}
	switch op {
	case "propose":
		if pm.isInvalidProposal(f[4:]) {
			csend(fmt.Sprintf("uh, %s, the format for proposals is:", nick))
			csendm(proposalFormat)
			return
		}
		if p%int64(len(pm.players)) != int64(talker.id) {
			rsp := nick + ": you can only use proposal numbers that are "
			rsp += fmt.Sprintf("%d modulo %d", talker.id, len(pm.players))
			csend(rsp)
			csend("everybody ignore " + nick + "'s proposal, please")
			csend("It's like that didn't happen.")
			return
		}
		oldp := pm.players[nick].lProposed
		if oldp > p {
			csend(fmt.Sprintf("%s: %s %d when you already proposed %d?",
				nick, "why are you proposing", p, pm.players[nick].lProposed))
			csend(fmt.Sprintf("folks, please forget %s mentioned %d",
				nick, p))
			return
		}
		pm.newProposed(pm.players[nick], p)
		for k, _ := range pm.players {
			if p > pm.players[k].seen {
				pm.players[k].seen = p
			}
		}
	case "promise":
		if pm.isInvalidPromise(f[4:]) {
			csend(fmt.Sprintf("well, %s, the format for promises is:", nick))
			csendm(promiseFormat)
			return
		}
		if !pm.wasProposed(p) {
			csend(fmt.Sprintf("hey, %s!  %d was never proposed", nick, p))
			return
		}
		pl := pm.players[nick]
		if pl.min > p {
			csend(fmt.Sprintf("%s has already promised not to accept proposals below %d",
				nick, pl.min))
			csend(fmt.Sprintf("so ... although not illegal, it's weird to promise for %d",
				p))
		} else {
			pl.min = p
		}
		// XXXtodo: 
		//   record promised values
	case "set":
		q, accepted, why := pm.quorumPromised(nick, p)
		if !q {
			csend(fmt.Sprintf("%s can't set a value until a majority has promised on proposal %d", nick, p))
			csend("  " + why)
			return
		}
		if accepted.val != nil {
			csend(fmt.Sprintf("%s, you have to set the value below, because it was accepted by %s as proposal %d", nick, accepted.acceptor, accepted.proposal))
			csend("\""+*accepted.val+"\"")
		} else {
			s := strings.Join(f[4:], " ")
			pm.players[nick].lSet = &s
			// now it's up to the acceptors to accept
		}
	case "accept":
		val := strings.Join(f[4:], " ")
		if !pm.wasSet(p, val) {
			csend(fmt.Sprintf("%s can't accept value for proposal %d that was never set",
				nick, p))
			return
		}
		a := pm.players[nick]
		if a.min > p {
			csend(fmt.Sprintf("%s, you promised not to accept any proposal less than %d", a.min))
			return
		}
		if a.aVal != nil && a.pAccepted < p && *a.aVal != val {
			csend(fmt.Sprintf("%s, you already accepted a value for proposal %d, with the value below:",
				nick, a.pAccepted))
			csend("  \"" + *a.aVal + "\"")
			csend("you can't accept a different lower-proposal-number value")
			return
		}
		a.aVal = &val
	default:
		csend("unknown operation: " + op)
	}
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
			pm.players[nick] = &Player{
				id: id,
			}
			newgame()
		}
	case "PART":
		if _, present := pm.players[nick]; present {
			delete(pm.players, nick)
			i := 0
			for _, v := range pm.players {
				v.id = i
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
		players: make(map[string]*Player),
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

// test IRC client uses https://github.com/husio/go-irc.git
// based on examples/client.go in go-irc
//
// GOPATH="$HOME"/git/go-irc go build ircctest.go

package main

import (
	"bufio"
	"flag"
	"fmt"
	"irc"
	"log"
	"os"
	"strings"
)

var server *string = flag.String("server", "irc.freenode.net", "IRC server address")
var port *int = flag.Int("port", 6667, "IRC server port")
var nick *string = flag.String("nick", "go-irc-client", "Nickname")

type PMod struct {
	ircchan string
}

func (pm *PMod) handle(send func(string), m string) bool {
	cont := true
	search := " PRIVMSG #" + pm.ircchan + " :"
	i := strings.Index(m, search)
	log.Printf("i:%d", i)
	if i > 0 {
		log.Printf("m[i+1:] \"%s\"", m[i+1:])
	}
	if i > 0 && i+len(search) < len(m) {
		switch m[i+len(search):] {
		case "foo":
			send("PRIVMSG #" + pm.ircchan + " :bar")
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
	pm := &PMod{"pmodtesting"}

	quit := make(chan bool)
	send("NICK " + *nick)
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

// The gorilla websocket server draws upon the module author's example
// at this URL: http://gary.burd.info/go-websocket-chat

package main

import (
	"code.google.com/p/xsrftoken"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
	htpt "html/template"
	"log"
	"net"
	"net/http"
	ttpt "text/template"
)

// Members have to be capitalized to get marshalled by json.
type Message struct {
	Csrf_token string
	Valid      bool
}

type Index struct {
	Welcome string
	Csrf    string
}

// XXX: Load from required source external to the source code
//  for production use.
const siteSecret = "Loaded from site config file."

var store = sessions.NewCookieStore([]byte(siteSecret))

func serveIndex(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	if session.IsNew {
		log.Printf("new session from %v", r.RemoteAddr)
		// store stuff, e.g., session.Values["answer"] = 42
		session.Save(r, w)
	}
	index, err := htpt.ParseFiles("index.html")
	if err != nil {
		w.Write([]byte(err.Error()))
	}
	err = index.Execute(w, &Index{Welcome: "Hello.", Csrf: csrf(r)})
	if err != nil {
		w.Write([]byte(err.Error()))
	}
}

const port = ":8181"
const wsURL = "ws://127.0.0.1" + port + "/ws"

func serveJs(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	if session.IsNew {
		log.Printf("reject no-session js request from %v",
			r.RemoteAddr)
		return
	}
	index, err := ttpt.ParseFiles("shform.js")
	if err != nil {
		log.Printf("template parse error: %s", err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/javascript")
	err = index.Execute(w, wsURL)
	if err != nil {
		log.Printf("template execute error: %s", err.Error())
		return
	}
}

func csrf(r *http.Request) string {
	user := r.RemoteAddr
	action := r.Method + r.URL.Path
	return xsrftoken.Generate(siteSecret, user, action)
}

func csrf_ok(r *http.Request, token string) bool {
	user := r.RemoteAddr
	action := r.Method + r.URL.Path
	return xsrftoken.Valid(token, siteSecret, user, action)
}

func serveDbg(w http.ResponseWriter, r *http.Request) {
	token := csrf(r)
	valid := csrf_ok(r, token)
	msg := Message{token, valid}
	buf, err := json.Marshal(msg)
	if err == nil {
		w.Write(buf)
	} else {
		w.Write([]byte(err.Error()))
	}
}

type hub struct {
	// The keys are the registered ws connections.
	connections map[*wsconn]bool

	// Outgoing messages to the clients.
	share chan UpdateMsg

	// Note clients through this channel.
	register chan *wsconn

	// Forget clients through this channel.
	unregister chan *wsconn
}

// This data structure is used to route messages.
var h = hub{
	share:       make(chan UpdateMsg),
	register:    make(chan *wsconn),
	unregister:  make(chan *wsconn),
	connections: make(map[*wsconn]bool),
}

func (h *hub) run() {
	log.Print("hub runs")
	for {
		select {
		case c := <-h.register:
			log.Printf("register ws client %v", c.ws.RemoteAddr())
			h.connections[c] = true
		case c := <-h.unregister:
			log.Printf("unregister ws client %v", c.ws.RemoteAddr())
			delete(h.connections, c)
			close(c.send)
		case m := <-h.share:
			log.Printf("hub shares %s", string(m.data))
			for c := range h.connections {
				if c.ws.RemoteAddr() == m.origin {
					log.Printf("hub skips %v",
						m.origin)
					continue
				}
				select {
				case c.send <- m.data:
					log.Printf("hub sent (%s) to %v",
						string(m.data),
						c.ws.RemoteAddr())
				default:
					log.Printf("hub removes %v",
						c.ws.RemoteAddr())
					delete(h.connections, c)
					close(c.send)
					go c.ws.Close()
				}
			}
		}
	}
}

type wsconn struct {
	ws   *websocket.Conn
	send chan []byte
}

type UpdateMsg struct {
	origin net.Addr
	data   []byte
}

func (c *wsconn) reader() {
	log.Print("reader start")
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			log.Print("reader exits on error")
			break
		}
		log.Print("reader gets message: %s", string(message))
		h.share <- UpdateMsg{c.ws.RemoteAddr(), message}
	}
	log.Print("reader close ws")
	c.ws.Close()
}

func (c *wsconn) writer() {
	log.Print("writer start")
	for message := range c.send {
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Print("writer error")
			break
		}
	}
	log.Print("writer close ws")
	c.ws.Close()
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	log.Print("serveWs")
	session, _ := store.Get(r, "session-name")
	if session.IsNew {
		log.Printf("rejecting ws without session from %v",
			r.RemoteAddr)
		return
	}
	ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if _, ok := err.(websocket.HandshakeError); ok {
		log.Printf("rejecting ws with bad handshake from %v",
			r.RemoteAddr)
		http.Error(w, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		log.Printf("serveWs error: %s", err.Error())
		return
	}
	log.Printf("serveWs ws connection from %v", r.RemoteAddr)
	c := &wsconn{send: make(chan []byte, 256), ws: ws}
	h.register <- c
	defer func() { h.unregister <- c }()
	go c.writer()
	c.reader()
}

// http://www.gorillatoolkit.org/pkg/mux
func main() {
	go h.run()
	r := mux.NewRouter()
	r.HandleFunc("/dbg", serveDbg)
	r.HandleFunc("/", serveIndex)
	r.HandleFunc("/ws", serveWs)
	r.HandleFunc("/shform.js", serveJs)
	// https://groups.google.com/forum/#!topic/gorilla-web/uspFHanLI3s
	r.PathPrefix("/pub/").
		Handler(http.StripPrefix("/pub/",
		http.FileServer(http.Dir("pub/"))))
	http.Handle("/", r)
	http.ListenAndServe(port, nil)
}

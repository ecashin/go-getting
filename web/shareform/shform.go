package main

import (
	"code.google.com/p/xsrftoken"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"html/template"
	"log"
	"net/http"
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
	index, err := template.ParseFiles("index.html")
	if err != nil {
		w.Write([]byte(err.Error()))
	}
	err = index.Execute(w, &Index{Welcome: "Hello.", Csrf: csrf(r)})
	if err != nil {
		w.Write([]byte(err.Error()))
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

// http://www.gorillatoolkit.org/pkg/mux
func main() {
	r := mux.NewRouter()
	r.HandleFunc("/dbg", serveDbg)
	r.HandleFunc("/", serveIndex)
	// https://groups.google.com/forum/#!topic/gorilla-web/uspFHanLI3s
	r.PathPrefix("/pub/").
		Handler(http.StripPrefix("/pub/",
		http.FileServer(http.Dir("pub/"))))
	http.Handle("/", r)
	http.ListenAndServe(":8181", nil)
}

package main

import (
	"code.google.com/p/xsrftoken"
	"encoding/json"
	"github.com/gorilla/mux"
	"html/template"
	"net/http"
)

// Members have to be capitalized to get marshalled by json.
type Message struct {
	Csrf_token string
	Valid      bool
}

type Index struct {
	Welcome string
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	index, err := template.ParseFiles("index.html")
	if err != nil {
		w.Write([]byte(err.Error()))
	}
	err = index.Execute(w, &Index{Welcome: "Hello."})
	if err != nil {
		w.Write([]byte(err.Error()))
	}
}

func serveDbg(w http.ResponseWriter, r *http.Request) {
	key := "shform No One Gonna Guess Dis"
	user := r.RemoteAddr
	action := r.Method + r.URL.Path
	csrf := xsrftoken.Generate(key, user, action)
	valid := xsrftoken.Valid(csrf, key, user, action)
	msg := Message{csrf, valid}
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
	http.Handle("/", r)
	http.ListenAndServe(":8181", nil)
}

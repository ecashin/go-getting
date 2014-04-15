package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

func serveHTTP(w http.ResponseWriter, r *http.Request) {
	buf, err := json.Marshal([]string{"Hi.", "There."})
	if err == nil {
		w.Write(buf)
	}
}

// http://www.gorillatoolkit.org/pkg/mux
func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", serveHTTP)
	http.Handle("/", r)
	http.ListenAndServe(":8181", nil)
}

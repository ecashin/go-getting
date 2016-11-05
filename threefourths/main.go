// Try using the interesting prototype described at URL below.
// https://beacon.nist.gov/home

package main

import (
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

const (
	NIST_RANDOM = "https://beacon.nist.gov/rest/record/last"
	N_ROUNDS    = 10
)

type Record struct {
	XMLName    xml.Name `xml:"record"`
	Version    string   `xml:"version"`
	Freq       string   `xml:"frequency"`
	Ts         int      `xml:"timeStamp"`
	SeedVal    string   `xml:"seedValue"`
	PrevVal    string   `xml:"previousOutputValue"`
	SigVal     string   `xml:"signatureValue"`
	OutputVal  string   `xml:"outputValue"`
	StatusCode string   `xml:"statusCode"`
}

func nistURL(ts int) string {
	return fmt.Sprintf("https://beacon.nist.gov/rest/record/previous/%d", ts)
}

func nistBytes(url string, c chan Msg) {
	doc := nistDoc(url)
	v := &Record{}
	err := xml.Unmarshal(doc, &v)
	if err != nil {
		panic(err)
	}
	fmt.Println("debug: " + v.OutputVal)
	a, err := hex.DecodeString(v.OutputVal)
	if err != nil {
		panic(err)
	}
	for _, b := range a {
		msg := <-c
		if msg.cmd == "getByte" {
			msg.resp <- b
		} else {
			return
		}
	}

	// Recurse on previous timestamp for remaining rounds.
	nistBytes(nistURL(v.Ts-1), c)
}

func nistDoc(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return body
}

type Msg struct {
	cmd  string
	resp chan byte
}

func main() {
	c := make(chan Msg)
	go nistBytes(NIST_RANDOM, c)
	resp := make(chan byte)
	nRounds := N_ROUNDS
	if len(os.Args) > 1 {
		n, err := strconv.ParseInt(os.Args[1], 0, 32)
		if err != nil {
			panic(err)
		}
		nRounds = int(n)
	}
	for ; nRounds > 0; nRounds-- {
		c <- Msg{"getByte", resp}
		n := <-resp
		for j := 0; j < 8; j++ {
			fmt.Println(n & 1)
			n = n >> 1
		}
	}
	c <- Msg{"quit", nil}
}

// Try using the interesting prototype described at URL below.
// https://beacon.nist.gov/home

package main

import (
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strconv"
)

const (
	NIST_RANDOM = "https://beacon.nist.gov/rest/record/last"
	N_ROUNDS    = 1000
	MIN_CHOICES = 2
	MAX_CHOICES = 64
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
	//fmt.Println("debug: " + v.OutputVal)
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

func nistBits(url string, c chan Msg) {
	byteChan := make(chan Msg)
	resp := make(chan byte)
	go nistBytes(url, byteChan)
	nBits := 0
	var b byte
	for {
		msg := <-c
		if msg.cmd != "getBit" {
			byteChan <- Msg{"quit", nil}
			return
		}
		if nBits == 0 {
			byteChan <- Msg{"getByte", resp}
			b = <-resp
			nBits = 8
		}
		msg.resp <- b & 1
		b >>= 1
		nBits--
	}
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

func simOne(n int, c chan Msg) {
	totalFlips := 0
	trialFlips := int(math.Ceil(math.Log2(float64(n))))
	resp := make(chan byte)
trials:
	for {
		selection := 0
		for i := 0; i < trialFlips; i++ {
			c <- Msg{"getBit", resp}
			toss := <-resp
			totalFlips++
			selection <<= 1
			selection |= int(toss)
		}
		if selection < n {
			break trials
		} else {
			// fmt.Printf("debug: %d too large for %d\n", selection, n)
		}
	}
	fmt.Printf("%d %d\n", n, totalFlips)
}

func main() {
	c := make(chan Msg)
	go nistBits(NIST_RANDOM, c)
	nRounds := N_ROUNDS
	if len(os.Args) > 1 {
		n, err := strconv.ParseInt(os.Args[1], 0, 32)
		if err != nil {
			panic(err)
		}
		nRounds = int(n)
	}
	for n := MIN_CHOICES; n <= MAX_CHOICES; n++ {
		for round := 0; round < nRounds; round++ {
			simOne(n, c)
		}
	}
	c <- Msg{"quit", nil}
}

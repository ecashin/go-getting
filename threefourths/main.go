// Try using the interesting prototype described at URL below.
// https://beacon.nist.gov/home

package main

import (
	"encoding/hex"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
)

const (
	HEADER = "nOptions selection nFlips"
	NIST_RANDOM = "https://beacon.nist.gov/rest/record/last"
	N_ROUNDS    = 100
	MIN_CHOICES = 2
	MAX_CHOICES = 64
)

var useGeorge bool
var nRounds int

func init() {
	flag.BoolVar(&useGeorge, "g", false,
		"use George's method")
	flag.IntVar(&nRounds, "n", 10,
		"number of simulations")
}

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

func intFromBits(c chan Msg, nBits int) int {
	result := 0
	resp := make(chan byte)
	for i := 0; i < nBits; i++ {
		c <- Msg{"getBit", resp}
		bit := <-resp
		result = (result << 1) | int(bit)
	}
	return result
}

func simGeorge(n int, c chan Msg) {
	kMin := int(math.Ceil(math.Log2(float64(n))))

	bestK := MAX_CHOICES * 2
	oldW := math.MaxFloat64
	var w float64
	for k := kMin; k < MAX_CHOICES * 2; k++ {
		z := int(math.Pow(2, float64(k))) % n
		p := 1.0 - float64(z) / math.Pow(2, float64(k))
		w = float64(k) / p - float64(kMin)
		if w > oldW {
			break
		}
		bestK = k
		oldW = w
	}

	var selection int
	totalFlips := 0
	for {
		j := intFromBits(c, bestK)
		totalFlips += bestK
		q := int(math.Pow(2, float64(bestK))) / n
		if j < q * n {
			selection = j % n
			break
		}
	}
	fmt.Printf("%d %d %d\n", n, selection, totalFlips)
}

func simOne(n int, c chan Msg) {
	totalFlips := 0
	trialFlips := int(math.Ceil(math.Log2(float64(n))))
	var selection int
trials:
	for {
		selection = intFromBits(c, trialFlips)
		totalFlips += trialFlips

		if selection < n {
			break trials
		} else {
			// fmt.Printf("debug: %d too large for %d\n", selection, n)
		}
	}
	fmt.Printf("%d %d %d\n", n, selection, totalFlips)
}

func main() {
	flag.Parse()
	c := make(chan Msg)
	go nistBits(NIST_RANDOM, c)
	fmt.Println(HEADER)
	for n := MIN_CHOICES; n <= MAX_CHOICES; n++ {
		for round := 0; round < nRounds; round++ {
			if useGeorge {
				simGeorge(n, c)
			} else {
				simOne(n, c)
			}
		}
	}
	c <- Msg{"quit", nil}
}

package main

import (
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
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

func nistBytes(url string, nRounds int, c chan byte) {
	if nRounds == 0 {
		close(c)
		return
	}
	doc := nistDoc(url)
	v := &Record{}
	err := xml.Unmarshal(doc, &v)
	if err != nil {
		panic(err)
	}
	// fmt.Println("debug: " + v.OutputVal)
	a, err := hex.DecodeString(v.OutputVal)
	if err != nil {
		panic(err)
	}
	for _, b := range a {
		c <- b
	}

	// Recurse on previous timestamp for remaining rounds.
	nRounds--
	nistBytes(nistURL(v.Ts-1), nRounds, c)
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

func main() {
	c := make(chan byte)
	go nistBytes(NIST_RANDOM, N_ROUNDS, c)
	counts := []int{0, 0, 0}
	for n := range c {
		for j := 0; j < 8; j += 2 {
			m := n & 3
			if m != 3 {
				counts[m] += 1
			}
			n = n >> 2
		}
	}
	fmt.Println(counts)
}

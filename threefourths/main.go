package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	NIST_RANDOM = "https://beacon.nist.gov/rest/record/last"
)


type Record struct {
	XMLName xml.Name `xml:"record"`
	Version string `xml:"version"`
	Freq string `xml:"frequency"`
	Ts string `xml:"timeStamp"`
	SeedVal string `xml:"seedValue"`
	PrevVal string `xml:"previousOutputValue"`
	SigVal string `xml:"signatureValue"`
	OutputVal string `xml:"outputValue"`
	StatusCode string `xml:"statusCode"`
}

func randstr(x []byte) string {
	v := &Record{}
	err := xml.Unmarshal(x, &v)
	if err != nil {
		panic(err)
	}
	return v.OutputVal
}


func flipCoins(c chan string, done chan bool, s string) {
	for i := 0; i < len(s); i++ {
		n := s[i]
		for j := 0; j < 8; j++ {
			if n & 1 > 0 {
				c <- "H"
			} else {
				c <- "T"
			}
			n = n >> 1
		}
	}
}


func main() {
	resp, err := http.Get(NIST_RANDOM)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	c := make(chan string)
	done := make(chan bool)
	go flipCoins(c, done, randstr(body))
	for toss := range c {
		fmt.Println(toss)
	}
}

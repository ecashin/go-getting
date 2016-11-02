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



func main() {
	resp, err := http.Get(NIST_RANDOM)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	counts := []int{0,0,0}
	for i := 0; i < len(body); i++ {
		n := body[i]
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

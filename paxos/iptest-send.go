package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

var ipAddr string
var ipProto int

func init() {
	flag.StringVar(&ipAddr, "a", "(none)",
		"destination IP address")
	flag.IntVar(&ipProto, "p", -1,
		"IP protocol number")
}
func main() {
	flag.Parse()
	if ipAddr == "(none)" || ipProto == -1 {
		log.Panic("usage")
	}
	ra, err := net.ResolveIPAddr("ip4", ipAddr)
	if err != nil {
		log.Panic(err)
	}
	proto := fmt.Sprintf("ip:%d", ipProto)
	conn, err := net.DialIP(proto, nil, ra)
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()
	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadSlice('\n')
	for err == nil {
		_, err = conn.Write([]byte(line))
		if err != nil {
			log.Panic(err)
		}
		line, err = in.ReadSlice('\n')
	}
}

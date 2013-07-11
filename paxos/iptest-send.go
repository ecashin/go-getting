package main

import (
	"log"
	"net"
)

func main() {
	ra, err := net.ResolveIPAddr("ip4", "127.0.0.1")
	if err != nil {
		log.Panic(err)
	}
	conn, err := net.DialIP("ip:253", nil, ra)
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()
	_, err = conn.Write([]byte("Hello, hoaloha."))
	if err != nil {
		log.Panic(err)
	}
}

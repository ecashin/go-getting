package main

import (
	"log"
	"net"
)

func main() {
	ra, err := net.ResolveUDPAddr("udp4", "127.0.0.1:4568")
	if err != nil {
		log.Panic(err)
	}
	conn, err := net.DialUDP("udp4", nil, ra)
	if err != nil {
		log.Panic(err)
	}
	_, err = conn.Write([]byte("Hello, hoaloha."))
	if err != nil {
		log.Panic(err)
	}
}

package main

import (
	"log"
	"net"
)

func main() {
	la, err := net.ResolveUDPAddr("udp4", "127.0.0.1:4568")
	if err != nil {
		log.Panic(err)
	}
	conn, err := net.ListenUDP("udp4", la)
	if err != nil {
		log.Panic(err)
	}
	buf := make([]byte, 9999)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Panic(err)
		}
		log.Print(string(buf[:n]))
	}
}

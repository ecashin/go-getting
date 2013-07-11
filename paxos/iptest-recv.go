package main

import (
	"log"
	"net"
)

func main() {
	la, err := net.ResolveIPAddr("ip4", "127.0.0.1")
	if err != nil {
		log.Panic(err)
	}
	conn, err := net.ListenIP("ip:253", la)
	if err != nil {
		log.Panic(err)
	}
	buf := make([]byte, 9999)
	for {
		n, ra, err := conn.ReadFromIP(buf)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("received from %s: %s", ra.String(), string(buf[:n]))
	}
}

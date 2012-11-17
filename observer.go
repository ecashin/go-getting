/* observer.go - HTTP RPC server watches critical section owner
 *
 * This program is for checking whether critical section
 * entrances and exits are interleaved for different owners,
 * signalling an error if there's a problem.
 */

package main

import (
	"errors"
	"net"
	"log"
	"fmt"
	"net/rpc"
	"net/http"
)

var port = 8766
var owner *string	// owner has access to critical section

type Observer struct {}
func (r *Observer) EnterCS(host *string, reply *int) error {
	log.Printf("enter: %s\n", *host)
	if owner != nil {
		log.Printf("VIOLATION: %s\n", *host)
		return errors.New(*host + " enters when owner already present");
	}
	owner = host
	*reply = 1
	return nil
}
func (r *Observer) ExitCS(host *string, reply *int) error {
	log.Printf(" exit: %s\n", *host)
	if owner == nil {
		log.Printf("VIOLATION: %s\n", *host)
		return errors.New(*host + " exits when no owner present");
	}
	owner = nil
	*reply = 0
	return nil
}

func main() {
	rpc.Register(new(Observer))
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if e != nil {
		log.Fatal("listen error:", e)
	}
	log.Printf("serving on %d\n", port)
	http.Serve(l, nil)
}

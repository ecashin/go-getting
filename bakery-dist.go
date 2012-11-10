/* At the bakery, the threads show up at the counter and "take a
 * number".  They hold a "ticket" with their number printed on it.
 * The corresponding data structure is the "number" array, which is
 * indexed by thread id.
 *
 * In the verbose output, the line,
 *
 * 2:foo >            1:bar
 *
 * ... reads, "host foo has bakery ticket 2, and it's waiting
 * for the host bar with bakery ticket 1".
 *
 * This distributed version relies on DNS being configured for
 * the hosts.  Launch it with the "other" hosts on standard input,
 * but make sure that the hostname on each line matches the value 
 * returned by os.Hostname() on that host.  For example, if on
 * tolstoy, os.Hostname() returns "tolstoy.coraid.com", you have
 * to spell that out as one line on the standard input when running
 * this program on host "marino".
 *
 * You have to start the program within a few seconds on all the
 * hosts.  Their servers come up first, and then they wait for
 * all the hosts to check in before starting the bakery algorithm.
 *
 * I wonder whether it's a bug to loop in the undefined key order
 * of the hosts map instead of in their static order, which is
 * lexically by host name in this program.
 *
 * usage example:
 *
 *   for i in a b c; do echo "$i"; done > hosts.txt
 *   for i in a b c; do
 *     ssh "$i" "go run bakery-dist.go > log 2>&1" < hosts.txt &
 *   done
 *   wait
 *
 */

package main

import (
	"os"
	"bufio"
	"runtime"
	"fmt"
	"time"
	"strings"
	"math/rand"
	"net"
	"net/rpc"
	"log"
	"net/http"
)

var nNodes int
type Customer struct {
	choosing bool
	number int
	name string
}
var me *Customer
var nIters = 500
var port = 8766

var liveHosts map[string]bool

func (t *Customer) BakeryNumber(requestor *string, reply *int) error {
	log.Printf("sending my number %d to %s\n", me.number, *requestor)
	*reply = me.number
	return nil
}
func (t *Customer) IsChoosing(requestor *string, reply *int) error {
	fmt.Printf("sending choosing{%v} to %s\n", me.choosing, *requestor)
	if me.choosing {
		*reply = 1
	} else {
		*reply = 0
	}
	return nil
}
func (t *Customer) HostUp(requestor *string, _ *int) error {
	log.Printf("%s is up\n", *requestor)
	liveHosts[*requestor] = true
	return nil
}
func (t *Customer) HostDown(requestor *string, _ *int) error {
	log.Printf("%s is down\n", *requestor)
	liveHosts[*requestor] = false
	return nil
}

func critical_section() {
	status(me.name, "critical")
	// hog the resource for a while
	time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond)
	status(me.name, "!critical")
}

func numOfHost(host string) int {
	if ! liveHosts[host] {
		return 0
	}
	n := doRPC(host, "Customer.BakeryNumber", "getting number from")
	return n
}

func isChoosing(host string) bool {
	if ! liveHosts[host] {
		return false
	}
	n := doRPC(host, "Customer.IsChoosing", "getting choosing state from")
	return n != 0
}

func maxNumber() int {
	m := 0
	for host, _ := range liveHosts {
		hnum := numOfHost(host)
		if hnum > m {
			m = hnum
		}
	}
	return m
}

// sort based on number order first, then participant (by-name) order
func lt(na int, sa string, nb int, sb string) bool {
	if na < nb {
		return true
	} else if na > nb {
		return false
	}
	return sa < sb
}

func status(stuff ...interface{}) {
	c := []int { 20, 12, }
	result := ""
	for i := 0; i < len(stuff); i++ {
		s := fmt.Sprint(stuff[i])
		if i < len(c) {
			f := fmt.Sprintf("%%-%ds", c[i])
			s = fmt.Sprintf(f, s)
		}
		result += s
	}
	fmt.Println(result)
}

func pair(a int, b string) string {
	return fmt.Sprintf("%d:%s", a, b)
}

func bakeryFun() {
	fmt.Printf("me{%s}\n", me)
	for i := 0; i < nIters; i++ {
		status(me.name, "choosing")
		me.choosing = true
		me.number = 1 + maxNumber()
		status(me.name, "number", me.number)
		me.choosing = false
		status(me.name, "!choosing")
		j := 0
		for host, _ := range liveHosts {
			choosing := true
			for choosing {
				choosing = isChoosing(host)
				if choosing {
					status(me.name, "waitchoose", j)
				}
			}
			n := numOfHost(host)
			for n != 0 && lt(n, host, me.number, me.name) {
				waiter := pair(me.number, me.name)
				winner := pair(n, host)
				status(waiter, ">", winner)
				n = numOfHost(host)
			}
		}
		critical_section()
		status(me.name, "number", 0)
		me.number = 0
	}
}

func srv() {
	rpc.Register(me)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if e != nil {
		log.Fatal("listen error:", e)
	}
	log.Println("serving")
	http.Serve(l, nil)
}

func waitHosts(which bool) {
	s := "up"
	if ! which {
		s = "down"
	}
	fmt.Printf("waiting for hosts to be %s\n", s)
	for true {
		done := true
		for _, v := range liveHosts {
			if v != which {
				done = false
			}
		}
		if done {
			return
		}
		time.Sleep(time.Duration(10 * time.Microsecond))
	}
}
func doRPC(host, method, description string) int {
	log.Printf("RPC to %s, method %s\n", host, method)
	client, err := rpc.DialHTTP("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Fatal("dialing ", host, ": ", err)
	}
	var result int
	err = client.Call(method, &me.name, &result)
	if err != nil {
		log.Fatal(description, " ", host, ": ", err)
	}
	client.Close()
	return result
}
func waitLive() {
	for k, _ := range liveHosts {
		doRPC(k, "Customer.HostUp", "announcing I'm up")
	}
	waitHosts(true)
	log.Println("all hosts are up")
}
func waitDead() {
	for k, _ := range liveHosts {
		doRPC(k, "Customer.HostDown", "announcing I'm down")
	}
	waitHosts(false)
	log.Println("all hosts are down")
}

func mpInit() {
	n := runtime.NumCPU()
	fmt.Printf("n CPUs: %d\n", n)
	runtime.GOMAXPROCS(n)
}

func main() {
	if len(os.Args) > 1 {
		_, err := fmt.Sscanf(os.Args[1], "%d", &nIters)
		if err != nil {
			panic(err);
		}
	}
	// any second command line argument disables debug output
	if len(os.Args) > 2 {
		null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			panic(err)
		}
		log.SetOutput(null)
	}
	rand.Seed(time.Now().UnixNano())
	mpInit()
	liveHosts = make(map[string]bool)

	in := bufio.NewReader(os.Stdin)
	myName, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	line, err := in.ReadSlice('\n')
	for err == nil {
		h := strings.TrimRight(string(line), "\n")
		if h != myName {
			liveHosts[h] = false
			nNodes++
		}
		line, err = in.ReadSlice('\n')
	}
	me = new(Customer)
	me.name = myName
	go srv()
	// Give the other servers a few seconds to come up.
	time.Sleep(time.Second * time.Duration(5))

	waitLive()
	bakeryFun()
	waitDead()
}

package main

import (
	"runtime"
	"fmt"
	"time"
	"math/rand"
)

var nCPUs int
var nIters = 5

/* The cs variable is here just to assert that mutual
 * exclusion is in effect.  It's not part of the bakery
 * algorithm.
 */
func critical_section(id int, cs *int) {
	*cs++	// this proc is in the critical section now
	if *cs > 1 {
		panic("mutual exclusion violated (1)")
	}
	fmt.Printf("%d in critical section\n", id)
	if *cs > 1 {
		panic("mutual exclusion violated (2)")
	}
	// hog the resource for a while
	time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
	if *cs > 1 {
		panic("mutual exclusion violated (3)")
	}
	*cs--	// done with critical section
}

func max(a[] int) int {
	m := a[0]	// panic on bad array
	for i := 1; i < len(a); i++ {
		if a[i] > m {
			m = a[i]
		}
	}
	return m
}

func proc(cs *int, id int, choosing, number[] int, done chan int) {
	y := func() { /* runtime.Gosched() */ }
	for i := 0; i < nIters; i++ {
		fmt.Printf("%d starts iteration %d\n", id, i)
		choosing[id] = 1
		number[id] = 1 + max(number)
		choosing[id] = 0
		for j := 0; j < len(choosing); j++ {
			for choosing[j] != 0 { y() }
			for number[j] != 0 && number[j] < number[id] { y() }
		}
		critical_section(id, cs)
		number[id] = 0
	}
	done<- id
}

func main() {
	nCPUs = runtime.NumCPU()
	var cs int = 0	// number procs in the critical section
	rand.Seed(time.Now().UnixNano())
	fmt.Printf("changing GOMAXPROCS from %d to %d\n",
		runtime.GOMAXPROCS(nCPUs), nCPUs)
	choosing := make([]int, nCPUs)
	number := make([]int, nCPUs)
	c := make(chan int)
	var i int
	for i = 0; i < nCPUs; i++ {
		go proc(&cs, i, choosing, number, c)
	}
	for i > 0 {
		fmt.Printf("%d is done\n", <-c)
		i--
	}
}

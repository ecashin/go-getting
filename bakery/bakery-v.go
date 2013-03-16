/* At the bakery, the threads show up at the counter and "take a
 * number".  They hold a "ticket" with their number printed on it.
 * The corresponding data structure is the "number" array, which is
 * indexed by thread id.
 *
 * In the verbose output, the line,
 *
 * 2:1 >            1:3
 *
 * ... reads, "thread with bakery ticket 2 is id 1, and it's waiting
 * for the thread with bakery ticket 1 (thread 3)".
 */

package main

import (
	"os"
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
	status(id, "critical")
	if *cs != 0 {
		panic("mutual exclusion violated")
	}
	*cs = 1
	// hog the resource for a while
	time.Sleep(time.Duration(rand.Intn(10)) * time.Microsecond)
	*cs = 0
	status(id, "!critical")
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

// sort based on number order first, then participant order
func lt(na, ia, nb, ib int) bool {
	if na < nb {
		return true
	} else if na > nb {
		return false
	}
	return ia < ib
}

func status(stuff ...interface{}) {
	c := []int { 4, 12, }
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

func pair(a, b int) string {
	return fmt.Sprintf("%d:%d", a, b)
}

func proc(cs *int, id int, choosing, number[] int, done chan int) {
	for i := 0; i < nIters; i++ {
		status(id, "choosing")
		choosing[id] = 1
		number[id] = 1 + max(number)
		status(id, "number", number[id])
		choosing[id] = 0
		status(id, "!choosing")
		for j := 0; j < len(choosing); j++ {
			for choosing[j] != 0 {
				status(id, "waitchoose", j)
			}
			for number[j] != 0 && lt(number[j], j, number[id], id) {
				waiter := pair(number[id], id)
				winner := pair(number[j], j)
				status(waiter, ">", winner)
			}
		}
		critical_section(id, cs)
		status(id, "number", 0)
		number[id] = 0
	}
	done<- id
}

func main() {
	if len(os.Args) > 1 {
		_, err := fmt.Sscanf(os.Args[1], "%d", &nIters)
		if err != nil {
			panic(err);
		}
	}		
	nCPUs = runtime.NumCPU()
	var cs int = 0	// a proc is in the critical section
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
		status(<-c, "DONE")
		i--
	}
}

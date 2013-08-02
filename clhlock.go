// implementation based on Herlihy's _The Art of Multiprocessor
// Programming_, section 7.5.2.

package main

import (
	"log"
	"math/rand"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"
)

type QNode struct {
	locked bool
}

// global state for the lock
type CLHLock struct {
	tail unsafe.Pointer	// *QNode manipulated by compare-and-swap
}

// thread-local state for the lock
type CLHLockThread struct {
	lk *CLHLock	// pointer to global lock state
	pred *QNode	// previous thread in implicit queue
	myNode *QNode	// changes to allow safe reclaim in lang like C
}

func (tlk *CLHLockThread) lock() {
	tlk.myNode.locked = true
	tlk.pred = (*QNode)(tlk.lk.tail)
	for !atomic.CompareAndSwapPointer(&tlk.lk.tail,
		unsafe.Pointer(tlk.pred),
		unsafe.Pointer(tlk.myNode)) {
		tlk.pred = (*QNode)(tlk.lk.tail)
	}
	for tlk.pred.locked {
		time.Sleep(time.Millisecond)
	}
}

func (tlk *CLHLockThread) unlock() {
	tlk.myNode.locked = false
	tlk.myNode = tlk.pred
}

func thread(lk *CLHLock, id int, done chan bool) {
	tlk := &CLHLockThread{lk, nil, &QNode{false}}
	f := "%d %-10s %s"
	out := "    OUT"
	log.Printf(f, id, "", "locking")
	tlk.lock()
	log.Printf(f, id, "IN", "did lock")
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	log.Printf(f, id, out, "unlocking")
	tlk.unlock()
	log.Printf(f, id, "", "did unlock")

	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

	log.Printf(f, id, "", "locking")
	tlk.lock()
	log.Printf(f, id, "IN", "did lock")
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	log.Printf(f, id, out, "unlocking")
	tlk.unlock()
	log.Printf(f, id, "", "did unlock")
	done <- true
}

func main() {
	clhlk := CLHLock{unsafe.Pointer(&QNode{false})}
	rand.Seed(time.Now().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())
	done := make(chan bool)
	n := 5
	for i := 0; i < n; i++ {
		go thread(&clhlk, i, done)
	}
	for i := 0; i < n; i++ {
		<- done
	}
}

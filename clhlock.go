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
	val int
}

// global state for the lock
type CLHLock struct {
	tail unsafe.Pointer	// *QNode manipulated by compare-and-swap
}

// thread-local state for the lock
type CLHLockThread struct {
	lk *CLHLock	// pointer to global lock state
	pred *QNode	// previous thread in implicit queue
	me *QNode	// "myNode" in Herlihy
}

func (tlk *CLHLockThread) lock() {
	tlk.me.locked = true
	tlk.pred = (*QNode)(tlk.lk.tail)
	for !atomic.CompareAndSwapPointer(&tlk.lk.tail,
		unsafe.Pointer(tlk.pred),
		unsafe.Pointer(&tlk.me)) {
		tlk.pred = (*QNode)(tlk.lk.tail)
	}
	if tlk.pred != nil {
		for tlk.pred.locked {
			time.Sleep(time.Millisecond)
		}
	}
}

func (tlk *CLHLockThread) unlock() {
	tlk.me.locked = false
	tlk.me = tlk.pred
}

func thread(lk *CLHLock, id int, done chan bool) {
	tlk := &CLHLockThread{lk, nil, &QNode{false, 0}}
	log.Printf("goroutine %d locking", id)
	tlk.lock()
	log.Printf("goroutine %d did lock", id)
	time.Sleep(time.Duration(rand.Intn(400)) * time.Millisecond)
	log.Printf("goroutine %d unlocking", id)
	tlk.unlock()
	log.Printf("goroutine %d did unlock", id)
	done <- true
}

func main() {
	var clhlk CLHLock
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

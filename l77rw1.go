/* Implementation of first concurrent readers/writer algorithm from
 * Leslie Lamport's 1977 paper, "Concurrent Reading and Writing."
 *
 * The algorithm works even when the data is read and written
 * without atomicity.  So I'm using an array to simulate more
 * complex distributed data structures.
 *
 * I can imagine, for example, a layout of data where v1 is in
 * one sector, datum is in a series of sectors, and v2 is in a
 * trailing sector.  Concurrent readers will get a consistent
 * state of the datum even though writers can always write it
 * without locking.
 */

package main

import (
	"fmt"
	"runtime"
	"math/rand"
	"time"
)

// this "variable" is a silly non-atomic datum for demonstration
type variable [4]uint8
var v1, v2, datum variable

func readLR(v *variable) uint32 {
	var n uint32
	len := len(*v)
	for i, _ := range *v {
		sh := uint((len - 1 - i) * 8)
		n |= uint32(v[i]) << sh
	}
	return n
}

func readRL(v *variable) uint32 {
	var n uint32
	len := len(*v)
	for i, _ := range *v {
		sh := uint((i) * 8)
		n |= uint32(v[len - 1 - i]) << sh
	}
	return n
}

func writeLR(v *variable, n uint32) {
	len := len(*v)
	for i, _ := range *v {
		sh := uint(len - i - 1) * 8
		v[i] = uint8(n >> sh)
	}
}

func writeRL(v *variable, n uint32) {
	len := len(*v)
	for i, _ := range *v {
		sh := uint(i) * 8
		v[len - i - 1] = uint8(n >> sh)
	}
}

func show(v *variable) {
	s := "v: { "
	for i, _ := range *v {
		s += fmt.Sprintf("0x%x ", v[i])
	}
	fmt.Printf("%s }\n", s)
}

const iters = 10

func pause() {
	time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
}

func reader(id int, c chan int) {
	for i := 0; i < iters; i++ {
		for {
			m := readLR(&v2)
			d := readLR(&datum)
			n := readRL(&v1)
			if n == m {
				break
			} else {
				fmt.Printf("%d read 0x%x v1:0x%x v2:0x%x\n",
					id, d, n, m)
			}
			runtime.Gosched()
//			pause()
		}
		pause()
	}
	c<- id
}

func writer(id int, c chan int) {
	for i := 0; i < iters; i++ {
		n := readLR(&v1)
		n++
		writeLR(&v1, n)
		fmt.Printf("write 0x%x\n", uint32(id) + n)
		writeLR(&datum, uint32(id) << uint(16) | n)
		writeRL(&v2, n)
		pause()
	}
	c<- id
}

const nReaders = 4

func main() {
	nCPUs := runtime.NumCPU()
	runtime.GOMAXPROCS(nCPUs)
	rand.Seed(time.Now().UnixNano())
	c := make(chan int)
	for i := 0; i < nReaders; i++ {
		go reader(i, c)
	}
	go writer(nReaders, c)
	wait := nReaders + 1
	for wait > 0 {
		fmt.Printf("%d exited\n", <-c)
		wait--
	}
}

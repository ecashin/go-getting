
PROGS = demo

.PHONY: test

all: $(PROGS)

clean:
	rm -f $(PROGS)

demo: demo.go src/intarr/intarr.go src/selsort/selsort.go
	GOPATH=`pwd` go build $<

test:
	GOPATH=`pwd` go test tree/rbtree
	GOPATH=`pwd` go test bubblesort
	GOPATH=`pwd` go test mergesort
	GOPATH=`pwd` go test selsort
	GOPATH=`pwd` go test inssort

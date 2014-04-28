
PROGS = demo

.PHONY: test

all: $(PROGS)

clean:
	rm -f $(PROGS)

demo: demo.go src/intarr/intarr.go src/selsort/selsort.go
	GOPATH=`pwd` go build $<

test:
	GOPATH=`pwd` go test inssort
	GOPATH=`pwd` go test selsort

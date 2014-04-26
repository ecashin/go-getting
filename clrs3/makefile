
PROGS = demo

.PHONY: test

all: $(PROGS)

clean:
	rm -f $(PROGS)

demo: demo.go src/intarr/intarr.go src/inssort/inssort.go
	GOPATH=`pwd` go build $<

test:
	GOPATH=`pwd` go test inssort

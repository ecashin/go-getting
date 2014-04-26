
PROGS = insertion_sort

all: $(PROGS)

clean:
	rm -f $(PROGS)

insertion_sort: insertion_sort.go src/intarr/intarr.go
	GOPATH=`pwd` go build $<

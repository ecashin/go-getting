
PROGS = insertion_sort

all: $(PROGS)

clean:
	rm -f $(PROGS)

%: %.go
	GOPATH=`pwd` go build $<

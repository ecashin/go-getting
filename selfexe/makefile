selfexe: selfexe.go payloadbin.go
	go build $^

.PHONY: clean realclean

clean:
	rm -f selfexe payload payloadbin.go

realclean: clean
	rm -f *~

payloadbin.go: payload
	{ \
	  echo 'package main'; \
	  echo 'var Bin  = `` +'; \
	  gzip < $< | xxd -ps -c 2000 | \
	  sed 's|\([a-f0-9][a-f0-9]\)|\\x\1|g' | \
	  sed 's|^|"|' | sed 's|$$|" + |'; \
	  echo ' ``'; \
	} > $@

%: %.go
	go build $<

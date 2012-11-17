go-getting
==========

Exercises in learning the Go programming language


bakery.go - Go implementation of 1974 Lamport bakery algorithm

  Leslie Lamport, author of the LaTeX macropackage for Don Knuth's TeX
  typesetting engine, is also famous for his often-cited papers on
  distributed computing.

  In his collection of papers, he has reflections written about each.
  His estimation of the bakery algorithm as a distributable mutual
  exclusion mechanism is that it has been underestimated for years.

  One nice property is that it resists the failure of a participant
  (as long as the participant doesn't use the exclusive resource while
  "failed").  Another is that writes to any single location are
  performed exclusively by a single participant, but a single location
  is read by multiple participants.

  http://research.microsoft.com/en-us/um/people/lamport/pubs/pubs.html#bakery

  The paper has a really simple listing that translates easily into
  code.  My program's "proc" function mirrors the paper's pseudocode.

  http://research.microsoft.com/en-us/um/people/lamport/pubs/bakery.pdf

  This Go program has the different participants on the same machine,
  and I learned that if I set the number of Go processes to the number
  of CPUs, the busy waiting in the algorithm works great.  If Go has
  to multiplex the goroutines onto shared O.S. threads, then they're
  acting as coroutines, so the runtime.Gosched() call is needed in the
  busy wait for the coroutines to yield the CPU.

bakery-v.go - Verbose Go implementation of 1974 Lamport bakery algorithm

bakery-dist.go - Distributed Go implementation of Lamport's bakery

  This distributed implementation shows how to use Go's HTTP
  RPC features.  In a production app where more performance was
  desired, connections could be reused.

selfwrite.go - Go program that writes its own source to stdout

linesrv.go - Simple web server demo

closure.go - Linked list iterator callback illustrates closure

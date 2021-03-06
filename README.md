go-getting
==========

Exercises in learning the Go programming language

2pc/node.go - Presume-abort two-phase commit

  This is a UDP-based proof-of-concept implementation of the simple
  and popular two-phase commit protocol.  It uses three processes as
  illustrated in the example usage at the top of the source.

android-apps/recommendations.go - Sam Rowe's Android app list

  When I tweeted about my new Android phone, my friend Sam Rowe sent
  me a list of recommended apps.  His email said, "Here's the fun
  version.  Let me know if you need the unfun version."

  More details are in the program.

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

observer.go - judge for correctness of bakery implementation

l77rw1.go - concurrent multiple readers / single writer

  based on Leslie Lamport's 1977 paper, "Concurrent Reading and Writing"

clhlock.go - Travis Craig and Magnussen, Landin, Hagersten lock demo

paxos/bpaxos.go - basic Paxos implementation

paxos/upaxos.go - IP-based unreliable broadcast Paxos demo

paxos/iptest-send.go - Send to IP broadcast

web/shareform/ - Demo of a multi-user web form edited concurrently

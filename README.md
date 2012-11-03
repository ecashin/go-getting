go-getting
==========

Exercises in learning the Go programming language


bakery.go - Go implementation of 1975 Lamport bakery algorithm

  Leslie Lamport, author of the LaTeX macropackage for Don Knuth's TeX
  typesetting engine, is also famous for his often-cited papers on
  distributed computing.

  In his collection of papers, he has reflections written about each.
  His estimation of the bakery algorithm as a distributable mutual
  exclusion mechanism is that it has been underestimated for years.

  http://research.microsoft.com/en-us/um/people/lamport/pubs/pubs.html#bakery

  It is pretty cool.  The paper has a really simple listing that
  translates easily into code.

  http://research.microsoft.com/en-us/um/people/lamport/pubs/bakery.pdf

  This Go program has the different participants on the same machine,
  and I learned that if I set the number of Go processes to the number
  of CPUs, the busy waiting in the algorithm works great.

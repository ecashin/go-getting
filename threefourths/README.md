# Three Forths

This simulation shows that when you flip a coin twice and throw out
all double-heads roles, you get a uniform distribution over the three
remaining possible outcomes.

It is rediculously elaborate, compared to the few lines of R that
would do the same thing, or compared to the intuitive or rigorous
thinking that one could do instead.  The reason is mostly because it
uses a hardware-based random number supply from NIST, which I thought
was kind of cool.

[https://beacon.nist.gov/home](https://beacon.nist.gov/home)

It also uses the XML encoding support in Go's core library, which I've
never had to do before.
# Three Forths

This simulation started out modeling the case where there are four
possible choices to make, and you're using two coin flips to make the
choice, throwing out all double-heads flips.

Now it simulates many numbers of choices.

The real reason for the Go implementation is mostly because it uses a
hardware-based random number supply from NIST, which I thought was
kind of cool but uses XML, giving me an excuse to try out the Go
core's XML support.

[https://beacon.nist.gov/home](https://beacon.nist.gov/home)

The `results.pdf` and `results-densities.pdf` plots were made with the
R code below.

    library(ggplot2)
    library(dplyr)
    
    d <- read.table(file="selection.log", header=F)
    names(d) <- c("n", "flips")
    d <- data.frame(d)
    qplot(x=n, y=flips, data=d) + geom_jitter()
    ggsave(filename="results.pdf")
    
    ggplot(d %>% filter(n > 8, n<20), aes(flips)) +
      geom_density() + facet_grid(n ~ ., scales="free_y")
    ggsave('results-densities.pdf')

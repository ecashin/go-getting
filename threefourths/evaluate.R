library(dplyr)
library(ggplot2)

plot.selection.counts <- function(fnam) {
    d <- tbl_df(read.table(fnam, header=T))
    ggplot(data=d %>%
          group_by(nOptions, selection) %>%
          mutate(count=n()) %>% distinct() %>%
          select(count),
          aes(x=selection, y=count)) +
        geom_point(alpha = 0.3) +
        facet_wrap(~ nOptions)
}

expected.flips <- function(fnam) {
    tbl_df(read.table(fnam, header=T)) %>%
        group_by(nOptions) %>%
        mutate(meanFlips=mean(nFlips)) %>%
        select(meanFlips) %>%
        distinct()
}

compare.flips <- function(naive, george) {
    qplot(x=x,
          y=y,
          data=data.frame(x=naive$nOptions,
                          y=100 * george$meanFlips / naive$meanFlips),
          ylab="percent of expected naive flips with George method",
          xlab="number of options")
}

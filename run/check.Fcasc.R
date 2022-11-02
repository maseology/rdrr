

library(ggplot2)
library(dplyr)



getBin <- function(fn) {
  fp <- paste0("M:/Peel/RDRR-PWRMM21/check/", fn)
  binsize <-  file.info(fp)$size/8
  binfile <- file(fp, "rb")
  bindata <- readBin(binfile, double(), n = binsize)
  data.frame(bindata)
}




getBin("Fcasc.bin") %>% ggplot(aes(bindata)) + geom_histogram() + labs(x="Fcasc")




k=100
getBin("STRC-dwngrad.bin") %>% ggplot(aes(bindata)) + 
  geom_histogram() +
  geom_function(fun = function(x) (1-exp(-k*x**2))*65000 , size=2, alpha=.65) +
  xlim(c(NA,.4)) +
  labs(title=paste0("(k = ",k,")"), x="land surface gradient")


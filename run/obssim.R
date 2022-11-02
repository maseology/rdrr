

library(dplyr)
library(tidyr)
library(dygraphs)
library(xts)








df.obs <- read.csv("M:/Peel/RDRR-PWRMM21/obs/784194-02HB013.csv") %>% #("M:/Peel/RDRR-PWRMM21/obs/1547060-02HB025.csv") %>%
  mutate(Date=as.Date(Date), Obs=Flow) %>%
  select(c(Date,Obs)) # [m³/s]


df.sim <- read.csv("wtrbdgt.csv") %>%
  mutate(Date=as.Date(date), Sim=hyd) %>%
  select(c(Date,Sim,Ya,AET,Recharge,DeltaStorage,Storage,MeanDm)) %>%
  group_by(Date) %>%
  summarise(Sim=mean(Sim), 
            Ya=sum(Ya), 
            AET=sum(AET), 
            Recharge=sum(Recharge), 
            DeltaStorage=sum(DeltaStorage), 
            Storage=mean(Storage),
            MeanDm=mean(MeanDm)) %>%
  ungroup()


df.mrg <- left_join(df.sim, df.obs, by="Date")












z=cbind(df.mrg$Obs, df.mrg$Sim, df.mrg$Ya, df.mrg$AET, df.mrg$Recharge, df.mrg$DeltaStorage, df.mrg$Storage, df.mrg$MeanDm)
x=xts(z, df.mrg$Date)
names(x) <- c("observed","simulated","AtmsYld","AET","Recharge","DeltaStorage","Storage","MeanDm")

dygraph(x) %>% #, main = "title goes here") %>% 
  dyRangeSelector(height = 20) %>%
  
  # dyBarSeries("AtmsYld", color="#1f78b4") %>%
  # dySeries("AET", strokeWidth = 2, color='green') %>%
  # dySeries("Recharge", strokeWidth = 2, color='grey') %>%
  # dySeries("DeltaStorage", strokeWidth = 2, color='cyan') %>%
  # dyAxis('y2', labelWidth=0) %>% #, valueRange = c(max(df.mrg$Ya)*1.5, 0)) %>%  
  
  dyBarSeries("AtmsYld", axis = 'y2', color="#1f78b4") %>%
  dySeries("AET", axis = 'y2', strokeWidth = 2, color='green') %>%
  dySeries("Recharge", axis = 'y2', strokeWidth = 2, color='grey') %>%
  dySeries("Storage", axis = 'y2', strokeWidth = 2, color='orange') %>%
  dySeries("DeltaStorage", axis = 'y2', strokeWidth = 2, color='cyan') %>%
  dyAxis('y2', labelWidth=0) %>% #, valueRange = c(max(df.mrg$Ya)*1.5, 0)) %>%

  dySeries("observed", strokeWidth = 2, color='blue') %>%
  dySeries("simulated", strokeWidth = 2, color='red') %>%
  dyLegend(width = 550)











z=cbind(df.mrg$Ya, df.mrg$AET, df.mrg$Recharge, df.mrg$DeltaStorage, df.mrg$Storage, df.mrg$MeanDm)
x=xts(z, df.mrg$Date)
names(x) <- c("AtmsYld","AET","Recharge","DeltaStorage","Storage","MeanDm")

dygraph(x) %>% #, main = "title goes here") %>% 
  dyRangeSelector(height = 20) %>%
  dyBarSeries("AtmsYld", color="#1f78b480") %>%
  dySeries("DeltaStorage", strokeWidth = 2, color='cyan') %>%
  dySeries("Storage", strokeWidth = 2, color='orange') %>%
  dySeries("AET", strokeWidth = 2, color='green') %>%
  dySeries("Recharge", strokeWidth = 2, color='brown') %>%
  
  # dyAxis('y2', labelWidth=0) %>% #, valueRange = c(max(df.mrg$Ya)*1.5, 0)) %>%

  dyLegend(width = 550)





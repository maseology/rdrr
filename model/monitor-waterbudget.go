package model

import (
	"fmt"
	"log"

	"github.com/maseology/mmio"
)

type tmonitor struct {
	sid                                int
	ys, ins, as, outs, gs, sto, bs, dm []float64
	dir                                string
}

func (tm *tmonitor) print(s0s, dm0 float64) {
	tmu.Lock()
	defer tmu.Unlock()
	defer gwg.Done()
	csvw := mmio.NewCSVwriter(fmt.Sprintf("%s%d.wbgt", tm.dir, tm.sid)) // subwatershed water budget file
	defer csvw.Close()
	if err := csvw.WriteHead("ys,ins,as,outs,sto,dsto,gs,bs,dm,ddm"); err != nil {
		log.Fatalf("%v", err)
	}

	dsto, ddm := tm.sto[0]-s0s, tm.dm[0]-dm0
	for i, y := range tm.ys {
		if i > 0 {
			dsto = tm.sto[i] - tm.sto[i-1]
			ddm = tm.dm[i] - tm.dm[i-1]
		}
		csvw.WriteLine(y, tm.ins[i], tm.as[i], tm.outs[i], tm.sto[i], dsto, tm.gs[i], tm.bs[i], tm.dm[i], ddm)
	}
}

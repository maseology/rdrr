package model

import (
	"fmt"
	"log"

	"github.com/maseology/mmio"
)

type tmonitor struct {
	sid                               int
	ys, ins, as, rs, gs, sto, bs, dm0 []float64
	dir                               string
}

func (tm *tmonitor) print() {
	tmu.Lock()
	defer tmu.Unlock()
	defer gwg.Done()
	csvw := mmio.NewCSVwriter(fmt.Sprintf("%s%d.wbgt", tm.dir, tm.sid)) // subwatershed water budget file
	defer csvw.Close()
	if err := csvw.WriteHead("ys,ins,as,rs,gs,sto,bs,dm0"); err != nil {
		log.Fatalf("%v", err)
	}
	for i, y := range tm.ys {
		csvw.WriteLine(y, tm.ins[i], tm.as[i], tm.rs[i], tm.gs[i], tm.sto[i], tm.bs[i], tm.dm0[i])
	}
}

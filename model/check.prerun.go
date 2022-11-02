package model

import (
	"fmt"

	"github.com/maseology/mmio"
)

func (dom *Domain) PreRunCheck(lus []*Surface, cxr map[int]int, xg, xm []int) {
	tt := mmio.NewTimer()

	ocids := dom.Strc.PrintAndCheck(dom.Dir)
	dom.Frc.PrintAndCheck(dom.Dir, ocids)

	fcasc, drel, dinc, bo, tm := make([]float64, dom.Nc), make([]float64, dom.Nc), make([]float64, dom.Nc), make([]float64, dom.Nc), make([]float64, dom.Nc)
	rzsto, detsto, fimp, perc := make([]float64, dom.Nc), make([]float64, dom.Nc), make([]float64, dom.Nc), make([]float64, dom.Nc)
	sDrel := make([]string, dom.Nc)
	xxg, xxm := make([]int32, dom.Nc), make([]int32, dom.Nc)
	for i, cid := range ocids {
		if _, ok := cxr[cid]; !ok {
			panic("cid error in checkmode")
		}
		surf := lus[cxr[cid]]
		fcasc[i], drel[i], dinc[i], tm[i] = surf.Fcasc, surf.Drel, surf.Dinc, surf.Tm
		bo[i] = func() float64 {
			if surf.Bo <= 0. {
				return -9999.
			}
			return surf.Bo
		}()
		rzsto[i], detsto[i] = surf.Hru.Sma.Cap, surf.Hru.Sdet.Cap
		fimp[i], perc[i] = surf.Hru.Fimp, surf.Hru.Perc
		xxg[i] = int32(xg[cxr[cid]])
		xxm[i] = int32(xm[cxr[cid]])
		sDrel[i] = fmt.Sprintf("%f,%d", drel[i], xxg[i])
	}
	// fmt.Printf(" model proportion at umin: %.1f%% \n", float64(ccasc)/float64(dom.Nc)*100.)
	writeFloats(dom.Dir+"/check/Fcasc.bin", fcasc)
	writeFloats(dom.Dir+"/check/Drel.bin", drel)
	mmio.WriteStrings(dom.Dir+"/check/Drel.txt", sDrel)
	writeFloats(dom.Dir+"/check/Dinc.bin", dinc)
	writeFloats(dom.Dir+"/check/Bo.bin", bo)
	writeFloats(dom.Dir+"/check/Tm.bin", tm)
	writeFloats(dom.Dir+"/check/rzsto.bin", rzsto)
	writeFloats(dom.Dir+"/check/detsto.bin", detsto)
	writeFloats(dom.Dir+"/check/Fimp.bin", fimp)
	writeFloats(dom.Dir+"/check/Perc.bin", perc)
	writeInts(dom.Dir+"/check/xg.indx", xxg)
	writeInts(dom.Dir+"/check/xm.indx", xxm)

	tt.Print("checkmode complete, see " + dom.Dir + "/check/")
}

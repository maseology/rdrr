package rdrr

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/maseology/mmio"
	"github.com/maseology/objfunc"
)

type monid struct{ batchnumber, samplenumber, isws, cellid int }

// ImportMCsamplesFromDir is a geneeral reader for sampling performed using evaluate.montecarlo.go
func (o *Obs) ImportMCsamplesFromDir(mcdir string, npar int, ts []time.Time, xr map[int]int) {
	var ok bool
	mons := make(map[monid][]float64)
	fps, _ := mmio.FileList(mcdir)
	chkcid := make(map[int]bool)
	for _, fp := range fps {
		if strings.HasSuffix(fp, ".samplespace.csv") {
			// print("-----------")
			lns, _ := mmio.ReadCSV(fp, 0)
			for i, ln := range lns {
				if i != int(ln[0]) {
					panic("wtf1")
				}
				if len(ln) != npar {
					panic("wtf2")
				}
			}
		} else if strings.HasSuffix(fp, "hyd.bin") {
			// print("=========hyd ")
		} else if strings.HasSuffix(fp, "sae.bin") {
			// print("=========sae ")
		} else if strings.HasSuffix(fp, "srch.bin") {
			// print("=========srch ")
		} else if strings.HasSuffix(fp, "sro.bin") {
			// print("=========sro ")
		} else if strings.HasSuffix(fp, ".bin") {
			qsim, _ := mmio.ReadBinaryFloats(fp)
			sp := strings.Split(mmio.FileName(fp, false), ".")
			mid := func() monid {
				batchnumber, _ := strconv.Atoi(sp[0])  // batch
				samplenumber, _ := strconv.Atoi(sp[1]) // sample
				isws, _ := strconv.Atoi(sp[3])         // (0-based) sws id
				cellid, _ := strconv.Atoi(sp[4])       // model cell id
				if cellid, ok = xr[cellid]; !ok {
					panic("not ok")
				}
				return monid{batchnumber, samplenumber, isws, cellid}
			}()

			mons[mid] = qsim
			if qobs, ok := (*o)[mid.cellid]; ok {
				oo, ss := todaily(qobs, qsim, ts)
				nse := objfunc.NSE(oo, ss)
				bias := objfunc.Bias(oo, ss)
				mobs, _ := objfunc.Meansd(oo)
				msim, _ := objfunc.Meansd(ss)
				fmt.Printf("  >> %d %s %f %f %f %f\n", len(qobs), sp, nse, bias, mobs, msim)
			} else {
				chkcid[mid.cellid] = true
				fmt.Printf("  WARNING: missing observations for observation point %d \n", mid.cellid)
			}
		}
		// println(fp)
	}
	fmt.Println(chkcid)
	fmt.Println(len(mons))
}

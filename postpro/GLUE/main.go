package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/maseology/goHydro/glue"
	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/postpro"
)

const (
	gdeffp = "M:/Peel/RDRR-PWRMM21/dat/elevation.real.uhdem.gdef" // "M:/OWRC-RDRR/owrc20-50a.uhdem.gdef"
	evalfp = "O:/PWRMM21.MC/summaryOF.csv"
	obsFP  = "M:/Peel/RDRR-PWRMM21/dat/elevation.real.uhdem.gauges_final.csv"
	minL   = 0.2
)

var (
	dtb   = time.Date(2010, 10, 1, 0, 0, 0, 0, time.UTC)
	dte   = time.Date(2020, 9, 30, 18, 0, 0, 0, time.UTC)
	intvl = 86400 / 4 * time.Second
)

func main() {
	tt := mmio.NewTimer()

	mcDir := mmio.GetFileDir(evalfp) + "/"
	gd, err := grid.ReadGDEF(gdeffp, true)
	if err != nil {
		log.Fatalf(" GLUE/main.go failed: %v\n", err)
	}

	// build model dates
	dts := func() []time.Time {
		t := make([]time.Time, int64(dte.Sub(dtb)/intvl)+1)
		ii := 0
		for dt := dtb; !dt.After(dte); dt = dt.Add(intvl) {
			t[ii] = dt
			ii++
		}
		return t
	}()

	// load observations
	obs := func() map[int]postpro.ObsColl {
		c, err := postpro.GetObservations(mcDir, obsFP)
		if err != nil {
			log.Fatalf(" postpro.GetObservations failed: %v", err)
		}
		return c
	}()

	// load results (after running \MC\main.go to create summaryOF.csv)
	realizations := readEvalCSV(evalfp)
	var ocol []map[int][]float64
	var gcol []map[int]float64
	var xr []int // behavioural realization ID to All realization ID
	func() {
		if _, ok := mmio.FileExists(mcDir + "glue-xr.gob"); !ok {
			// collect behavioural models
			txr, cnt := []int{}, 0
			for i, r := range realizations {
				if r.NSE < minL {
					continue
				}
				if r.KGE < minL {
					continue
				}
				cnt++
				txr = append(txr, i)
			}
			// remove duplicates, taking maximum likelihood (fuzzy union)
			eval, teval := map[string]float64{}, map[string]int{}
			for _, x := range txr {
				r := realizations[x]
				if v, ok := eval[r.Name]; !ok {
					eval[r.Name] = r.KGE
					teval[r.Name] = x
				} else {
					if v < r.KGE {
						eval[r.Name] = r.KGE
						teval[r.Name] = x
					}
				}
				xr = make([]int, 0, len(eval))
				for _, k := range teval {
					xr = append(xr, k)
				}
			}
			fmt.Printf("%d behavioural observations found, %d of %d retained\n", cnt, len(xr), len(realizations))

			ocol, gcol = make([]map[int][]float64, len(xr)), make([]map[int]float64, len(xr))
			for ii, i := range xr {
				ocol[ii], gcol[ii] = collectResults(mcDir + realizations[i].Name + ".tar.gz")
			}

			if err := saveGOBi(mcDir+"glue-xr.gob", xr); err != nil {
				log.Fatalf(" saveGOB failed: %v", err)
			}
			if err := saveGOBa(mcDir+"glue-ocol.gob", ocol); err != nil {
				log.Fatalf(" saveGOB failed: %v", err)
			}
			if err := saveGOB(mcDir+"glue-gcol.gob", gcol); err != nil {
				log.Fatalf(" saveGOB failed: %v", err)
			}

			tt.Lap("rdrr-MC postpro (collect to gob) complete.. exiting")
			// os.Exit(2)
		} else {
			var err error
			if xr, err = loadGOBi(mcDir + "glue-xr.gob"); err != nil {
				log.Fatalf(" loadGOB failed: %v", err)
			}
			if ocol, err = loadGOBa(mcDir + "glue-ocol.gob"); err != nil {
				log.Fatalf(" loadGOB failed: %v", err)
			}
			print(".")
			if gcol, err = loadGOB(mcDir + "glue-gcol.gob"); err != nil {
				log.Fatalf(" loadGOB failed: %v", err)
			}
			println(".")
		}
	}()

	func() map[int][]glue.GLUE {
		gQ := make(map[int][]glue.GLUE, len(obs)) // observation,timestep,glue array
		for m := range obs {                      // initialize
			if _, ok := ocol[0][m]; ok {
				gg := make([]glue.GLUE, len(ocol[0][m]))
				for i := range ocol[0][m] {
					gg[i] = make(glue.GLUE, len(xr))
				}
				gQ[m] = gg
			}
		}

		// collect
		for i, ir := range xr {
			r := realizations[ir]
			for m := range gQ {
				for ii, o := range ocol[i][m] {
					gQ[m][ii][i] = glue.GLUEi{Likelihood: r.KGE, Value: o}
				}
			}
		}
		// sort and collect bounds
		for m, v := range gQ {
			fobs := func() []float64 {
				obs := obs[m]
				fobs := make([]float64, len(dts))
				c := make(map[time.Time]float64, len(obs.T))
				for i, t := range obs.T {
					c[t] = obs.V[i]
				}
				dd := mmio.DayDate
				for i, t := range dts {
					if v, ok := c[dd(t)]; ok {
						fobs[i] = v
					} else {
						fobs[i] = 0.
					}
				}
				return fobs
			}()

			p5, p95 := make([]float64, len(v)), make([]float64, len(v))
			for ii := 0; ii < len(v); ii++ {
				sort.Sort(v[ii])
				p5[ii], p95[ii] = v[ii].P5o95()
			}

			sca := mmio.NewCSVwriter(mcDir + strconv.Itoa(m) + "-glue.csv")
			sca.WriteHead("dt,q,p5,p95")
			for i, t := range dts {
				sca.WriteLine(t.Format("2006-01-02 15:04:05"), fobs[i], p5[i], p95[i])
			}
			sca.Close()
			gQ[m] = v
		}

		return gQ
	}()

	func() []glue.GLUE { // initialize
		gG := make([]glue.GLUE, gd.Nact)
		cxr := gd.CellIndexXR()
		for cid := range gcol[0] {
			gG[cxr[cid]] = make(glue.GLUE, len(xr))
		}
		for i, ir := range xr {
			r := realizations[ir]
			for cid, val := range gcol[i] {
				gG[cxr[cid]][i] = glue.GLUEi{Likelihood: r.KGE, Value: val}
			}
		}

		p5, p95 := make([]float64, len(gG)), make([]float64, len(gG))
		for i := 0; i < len(gG); i++ {
			sort.Sort(gG[i])
			p5[i], p95[i] = gG[i].P5o95()
		}

		mmio.WriteBinary(mcDir+"GWE-p5-glue.real", p5)
		mmio.WriteBinary(mcDir+"GWE-p95-glue.real", p95)

		return gG
	}()

	tt.Lap("rdrr-MC postpro complete")
}

type Realization struct {
	Name           string
	StationID      int
	KGE, NSE, Bias float64
	Parameters     []float64
}

func readEvalCSV(filepath string) []Realization {
	f, err := os.Open(filepath)
	if err != nil {
		log.Fatalf("readEvalCSV failed: %v\n", err)
	}
	defer f.Close()

	chkerr := func(e error) {
		if e != nil {
			log.Fatalf(" readEvalCSV failed: %v\n", e)
		}
	}

	out := []Realization{}
	for rec := range mmio.LoadCSV(io.Reader(f)) {
		sid, err := strconv.Atoi(rec[1])
		chkerr(err)
		kge, err := strconv.ParseFloat(rec[2], 64)
		chkerr(err)
		nse, err := strconv.ParseFloat(rec[3], 64)
		chkerr(err)
		bias, err := strconv.ParseFloat(rec[4], 64)
		chkerr(err)

		pars := make([]float64, len(rec)-5)
		for i := 5; i < len(rec); i++ {
			pars[i-5], err = strconv.ParseFloat(rec[i], 64)
			chkerr(err)
		}

		r := Realization{
			Name:       rec[0],
			StationID:  sid,
			KGE:        kge,
			NSE:        nse,
			Bias:       bias,
			Parameters: pars,
		}
		out = append(out, r)
	}
	return out
}

func collectResults(tarfp string) (map[int][]float64, map[int]float64) {
	fmt.Printf(" extracting %s\n", tarfp)
	tmpdir, err := mmio.ExtractTarGZ(tarfp)
	defer mmio.DeleteDir(tmpdir)
	if err != nil {
		log.Fatalf(" ExtractTarGZ failed: %v", err)
		// fmt.Printf(" ExtractTarGZ failed: %v", err)
	}

	var o map[int][]float64 // [observationID][q timeseries]
	var g map[int]float64   // [cellID][values]

	// read distributed data
	func() {
		fps := mmio.FileListExt(tmpdir, ".rmap")
		if len(fps) == 0 {
			log.Fatalf(" ExtractTarGZ rmap read error: read bin files?? TODO\n")
		}
		for _, fp := range fps {
			if fp == tmpdir+"g.gwe.rmap" {
				if g, err = mmio.ReadBinaryRMAP(fp); err != nil {
					log.Fatalf(" ExtractTarGZ rmap (%s) read failed: %v", fp, err)
				}
			}
		}
	}()

	// read monitors
	func() {
		fps := mmio.FileListExt(tmpdir, ".cms")
		o = make(map[int][]float64, len(fps))
		for _, fp := range fps {
			fid, err := strconv.Atoi(mmio.FileName(fp, false))
			if err != nil {
				log.Fatalf(" filename error (cannot convert to number): %s: %v", mmio.FileName(fp, true), err)
			}

			qfid, err := mmio.ReadFloats(fp)
			if err != nil {
				log.Fatalf(" ReadFloats failed for %s: %v", fp, err)
			}

			o[fid] = qfid
		}
	}()

	return o, g
}

package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/maseology/mmio"
)

const (
	mcDir   = "S:/Peel/PWRMM21.MC/"                           // "C:/Users/Mason/Desktop/New folder/"
	obsFP   = "S:/Peel/elevation.real.uhdem.gauges_final.csv" //"M:/Peel/RDRR-PWRMM21/dat/elevation.real.uhdem.gauges_final.csv"
	jsonAPI = "https://api.oakridgeswater.ca/api/locnamsw?l="
	npar    = 4
)

var (
	dtb   = time.Date(2010, 10, 1, 0, 0, 0, 0, time.UTC)
	dte   = time.Date(2020, 9, 30, 18, 0, 0, 0, time.UTC)
	intvl = 86400 / 4 * time.Second
)

func main() {
	tt := mmio.NewTimer()
	defer tt.Lap("rdrr postpro complete")

	fmt.Println(" reading observation locations from: " + obsFP)

	// load observations
	colls := func() map[int]coll {
		var colls map[int]coll
		var err error
		if _, ok := mmio.FileExists(mcDir + "obs.gob"); !ok {
			colls, err = getObservations(obsFP)
			if err != nil {
				log.Fatalf(" getObservations failed: %v", err)
			}
			saveGob(colls, mcDir+"obs.gob")
		} else {
			colls, err = loadGob(mcDir + "obs.gob")
			if err != nil {
				log.Fatalf(" getObservations loadGob failed: %v", err)
			}
		}
		return colls
	}()

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

	// load model results
	csvw, lst, frst := mmio.NewCSVwriter(mcDir+"summaryOF.csv"), []string{}, true
	defer csvw.Close()
	for _, fp := range mmio.FileListExt(mcDir, ".gz") {
		c := collectEvals(fp, dts, colls)
		if c == nil {
			continue
		}
		// fmt.Println(c)

		// write header
		if frst {
			lst = make([]string, len(c)-npar)
			ii := 0
			hed := "mmid,m,grange,soildepth,kfact"
			for k := range c {
				if k == "m" || k == "grange" || k == "soildepth" || k == "kfact" {
					continue
				}
				lst[ii] = k
				hed += ",c" + k
				ii++
			}
			if err := csvw.WriteHead(hed); err != nil {
				log.Fatalf("WriteHead failed")
			}
			frst = false
		}

		row := make([]interface{}, len(c)+1)
		row[0] = mmio.FileName(fp, false)
		row[1] = c["m"]
		row[2] = c["grange"]
		row[3] = c["soildepth"]
		row[4] = c["kfact"]
		for i := npar + 1; i < len(row); i++ {
			row[i] = c[lst[i-npar-1]]
		}
		csvw.WriteLine(row...)
	}
}

func collectEvals(fp string, dts []time.Time, colls map[int]coll) map[string]float64 {
	fmt.Printf(" extracting %s\n", fp)
	tmpdir, err := mmio.ExtractTarGZ(fp)
	if err != nil {
		log.Fatalf(" ExtractTarGZ failed: %v", err)
	}
	defer mmio.DeleteDir(tmpdir)

	// read parameters of current realization
	if _, ok := mmio.FileExists(tmpdir + "params.txt"); !ok {
		fmt.Println("params.txt does not exist")
		return nil
	}
	sa, err := mmio.ReadTextLines(tmpdir + "params.txt")
	if err != nil {
		log.Fatalf(" ReadTextLines fail: %v", err)
	}
	par := make(map[string]float64, 4)
	for i := 2; i < len(sa); i++ {
		s := strings.Split(sa[i], "\t")
		par[s[0]], err = strconv.ParseFloat(s[1], 64)
		if err != nil {
			log.Fatalf("strconv.ParseFloat error: %s", sa[i])
		}
	}

	// read monitors
	fps := mmio.FileListExt(tmpdir, ".mon")
	o := make(map[string]float64, len(par)+len(fps))
	for k, v := range par {
		o[k] = v
	}
	for _, fp := range fps {
		sfid := mmio.FileName(fp, false)
		fid, err := strconv.Atoi(sfid)
		if err != nil {
			log.Fatalf(" filename error: %s: %v", mmio.FileName(fp, true), err)
		}
		if _, ok := colls[fid]; !ok {
			// fmt.Printf(" filename error: %s: fid not found\n", mmio.FileName(fp, true))
			continue
		}

		qfid, err := mmio.ReadFloats(fp)
		if err != nil {
			log.Fatalf(" ReadFloats failed for %s: %v", fp, err)
		}
		// n, kge, nse, bias := evaluate(dts, qfid, colls[fid])
		// fmt.Println(n, kge, nse, bias)
		_, kge, _, _ := evaluate(dts, qfid, colls[fid])
		o[sfid] = kge
	}
	return o
}

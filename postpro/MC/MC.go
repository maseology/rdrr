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
	mcDir   = "C:/Users/Mason/Desktop/New folder/"                             // "S:/Peel/PWRMM21.MC/" //
	obsFP   = "M:/Peel/RDRR-PWRMM21/dat/elevation.real.uhdem.gauges_final.csv" //"S:/Peel/elevation.real.uhdem.gauges_final.csv" //
	jsonAPI = "https://api.oakridgeswater.ca/api/locnamsw?l="
	npar    = 4
	minOF   = -1
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
	obsColls := func() map[int]obsColl {
		var c map[int]obsColl
		var err error
		if _, ok := mmio.FileExists(mcDir + "obs.gob"); !ok {
			c, err = getObservations(obsFP)
			if err != nil {
				log.Fatalf(" getObservations failed: %v", err)
			}
			saveGob(c, mcDir+"obs.gob")
		} else {
			c, err = loadGob(mcDir + "obs.gob")
			if err != nil {
				log.Fatalf(" getObservations loadGob failed: %v", err)
			}
		}
		return c
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

	// create output csv
	csvw := mmio.NewCSVwriter(mcDir + "summaryOF.csv")
	defer csvw.Close()
	if err := csvw.WriteHead("realization,station,KGE,NSE,bias,m,grange,soildepth,kfact"); err != nil {
		log.Fatalf("WriteHead failed")
	}

	// print model results
	for _, fp := range mmio.FileListExt(mcDir, ".gz") {
		rlz := mmio.FileName(mmio.FileName(fp, false), false)
		for _, c := range collectResults(fp, dts, obsColls) {
			if c.nse <= minOF {
				continue
			}
			csvw.WriteLine(rlz, c.fid, c.kge, c.nse, c.bias, c.par["m"], c.par["grange"], c.par["soildepth"], c.par["kfact"])
		}
	}
}

type stationResult struct {
	par            map[string]float64
	kge, nse, bias float64
	fid            int
}

func collectResults(tarfp string, dts []time.Time, obs map[int]obsColl) []stationResult {
	fmt.Printf(" extracting %s\n", tarfp)
	tmpdir, err := mmio.ExtractTarGZ(tarfp)
	if err != nil {
		log.Fatalf(" ExtractTarGZ failed: %v", err)
	}
	defer mmio.DeleteDir(tmpdir)

	// read parameters of current realization
	par := func() map[string]float64 {
		const parHead = 2 // n lines to skip
		par := make(map[string]float64, npar)
		if _, ok := mmio.FileExists(tmpdir + "params.txt"); !ok {
			fmt.Printf("params.txt does not exist in %s\n", tarfp)
			return nil
		}
		sa, err := mmio.ReadTextLines(tmpdir + "params.txt")
		if err != nil {
			log.Fatalf(" ReadTextLines failed in %s: %v", tarfp, err)
		}

		for i := parHead; i < len(sa); i++ {
			s := strings.Split(sa[i], "\t")
			par[s[0]], err = strconv.ParseFloat(s[1], 64)
			if err != nil {
				log.Fatalf("strconv.ParseFloat error in %s: %s", tarfp, sa[i])
			}
		}
		return par
	}()

	// read monitors
	fps := mmio.FileListExt(tmpdir, ".mon")
	o := make([]stationResult, len(fps))
	for i, fp := range fps {
		fid, err := strconv.Atoi(mmio.FileName(fp, false))
		if err != nil {
			log.Fatalf(" filename error (cannot convert to number): %s: %v", mmio.FileName(fp, true), err)
		}
		if _, ok := obs[fid]; !ok {
			o[i] = stationResult{fid: -9999, kge: minOF, nse: minOF}
			continue
		}

		qfid, err := mmio.ReadFloats(fp)
		if err != nil {
			log.Fatalf(" ReadFloats failed for %s: %v", fp, err)
		}

		_, kge, nse, bias := evaluate(dts, qfid, obs[fid])
		o[i] = stationResult{par: par, kge: kge, nse: nse, bias: bias, fid: fid}
	}
	return o
}

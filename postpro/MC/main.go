package main

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/maseology/mmio"
	pp "github.com/maseology/rdrr/postpro"
)

const (
	mcDir = "S:/Peel/PWRMM21.MC/"                           // "M:/Peel/RDRR-PWRMM21/PWRMM21.MC/"                               //
	obsFP = "S:/Peel/elevation.real.uhdem.gauges_final.csv" //"M:/Peel/RDRR-PWRMM21/dat/elevation.real.uhdem.gauges_final.csv" //
	npar  = 7
	minOF = -9999
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
	obsColls := func() map[int]pp.ObsColl {
		var c map[int]pp.ObsColl
		var err error
		c, err = pp.GetObservations(mcDir, obsFP)
		if err != nil {
			log.Fatalf(" postpro.GetObservations failed: %v", err)
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
	csvw, frst := mmio.NewCSVwriter(mcDir+"summaryOF.csv"), true
	defer csvw.Close()

	// print model results
	for _, fp := range mmio.FileListExt(mcDir, ".gz") {
		rlz := mmio.FileName(mmio.FileName(fp, false), false)
		for _, c := range collectResults(fp, dts, obsColls) {
			if c.pars == nil {
				continue
			}
			if frst {
				// writeHead(keys(c.par))
				shed := make([]string, len(c.pars)+1)
				shed[0] = "realization,station,KGE,NSE,bias"
				for i := 1; i <= len(c.pars); i++ {
					shed[i] = c.pars[i-1].name
				}
				if err := csvw.WriteHead(strings.Join(shed, ",")); err != nil {
					log.Fatalf("WriteHead failed")
				}
				frst = false
			}
			// if c.nse <= minOF {
			// 	continue
			// }
			sval := make([]float64, len(c.pars))
			for i := 0; i < len(c.pars); i++ {
				sval[i] = c.pars[i].value
			}
			csvw.WriteLine(rlz, c.fid, c.kge, c.nse, c.bias, sval)
		}
	}
}

type par struct {
	name  string
	value float64
}
type stationResult struct {
	pars           []par
	kge, nse, bias float64
	fid            int
}

func collectResults(tarfp string, dts []time.Time, obs map[int]pp.ObsColl) []stationResult {
	fmt.Printf(" extracting %s\n", tarfp)
	tmpdir, err := mmio.ExtractTarGZ(tarfp)
	if err != nil {
		log.Fatalf(" ExtractTarGZ failed: %v", err)
	}
	defer mmio.DeleteDir(tmpdir)

	// read parameters of current realization
	fpar := func() []par {
		const parHead = 2 // n lines to skip
		pars := make([]par, npar)
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
			flt, err := strconv.ParseFloat(s[1], 64)
			if err != nil {
				log.Fatalf("strconv.ParseFloat error in %s: %s", tarfp, sa[i])
			}
			pars[i-parHead] = par{s[0], flt}
		}
		return pars
	}()

	// read monitors
	fps := mmio.FileListExt(tmpdir, ".cms")
	o := make([]stationResult, len(fps))
	for i, fp := range fps {
		fid, err := strconv.Atoi(mmio.FileName(fp, false))
		if err != nil {
			// continue
			log.Fatalf(" filename error (cannot convert to number): %s: %v", mmio.FileName(fp, true), err)
		}
		if _, ok := obs[fid]; !ok {
			o[i] = stationResult{fid: -9999, kge: -math.MaxFloat64, nse: -math.MaxFloat64}
			continue
		}

		qfid, err := mmio.ReadFloats(fp)
		if err != nil {
			log.Fatalf(" ReadFloats failed for %s: %v", fp, err)
		}

		_, kge, nse, bias := evaluate(fp, dts, qfid, obs[fid])
		o[i] = stationResult{pars: fpar, kge: kge, nse: nse, bias: bias, fid: fid}
	}
	return o
}

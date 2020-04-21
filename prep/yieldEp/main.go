package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/im7mortal/UTM"
	"github.com/maseology/goHydro/grid"
	"github.com/maseology/goHydro/met"
	"github.com/maseology/goHydro/tem"
	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/prep"
)

const (
	gdefFP    = "S:/ormgp_rdrr/met/ORMGP_500a.gdef"
	gdefMdlFP = "S:/ormgp_rdrr/ORMGP_50_hydrocorrect.uhdem.gdef"
	demFP     = "S:/ormgp_rdrr/met/ORMGP_500.hdem"
	metFP     = "S:/ormgp_rdrr/met/ORMGP_500a_YCDB.met"
	metOutFP  = "S:/ormgp_rdrr/met/" //"S:/ormgp_rdrr/met/RDRR_500a_YCDB.met" //
	patm      = 101300.              // (constant) atmospheric pressure [Pa]
)

func main() {
	// check()

	var wg sync.WaitGroup
	defer wg.Wait()
	tt := mmio.NewTimer()
	defer tt.Print("prep complete!")

	gd, err := grid.ReadGDEF(gdefFP, true)
	if err != nil {
		log.Fatalf("%v", err)
	}
	nact := len(gd.Sactives)
	if nact <= 0 {
		log.Fatalf("error: grid definition requires active cells")
	}
	go func() {
		wg.Add(1)
		defer wg.Done()
		saveIntersect(metOutFP+"metIntersect.gob", gd)
	}()

	tem, err := tem.NewTEM(demFP)
	if err != nil {
		log.Fatalf("%v", err)
	}
	for _, i := range gd.Sactives {
		if tem.TEC[i].Z == -9999. {
			log.Fatalf("no elevation assigned to cell %d", i)
		}
	}

	hdr, dat, err := met.ReadBigMET(metFP, true)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if hdr.ESPG != 26917 { // UTM zone 17N
		log.Fatalf("TODO: ESPG not supported %d", hdr.ESPG)
	}

	// initialize
	cells, x := make([]prep.Cell, nact), hdr.WBDCxr()
	b, g, alpha, beta := .06142, .899, 1.3077261, -0.000361             // calibrated PE parameters
	tindex, ddfc, baseT, tsf := 0.009981, 1.794442, -2.035386, 0.211562 // calibrated snowmelt parameters
	for k, i := range gd.Sactives {
		latitude, _, err := UTM.ToLatLon(gd.Coord[i].X, gd.Coord[i].Y, 17, "", true)
		if err != nil {
			log.Fatalf(" prep error: %v -- (x,y)=(%f, %f); cid: %d\n", err, gd.Coord[i].X, gd.Coord[i].Y, i)
		}
		t := tem.TEC[i]
		cells[k] = prep.NewCell(latitude, math.Tan(t.G), math.Pi/2.-t.A, b, g, alpha, beta, tindex, ddfc, baseT, tsf)
	}

	if mmio.IsDir(metOutFP) {
		// build .gob
		ys, as := make([][]float64, nact), make([][]float64, nact) // [cell ID][date ID]
		for i := 0; i < nact; i++ {
			ys[i] = make([]float64, len(dat.T))
			as[i] = make([]float64, len(dat.T))
		}
		for k, dt := range dat.T {
			fmt.Println(dt)
			for i := 0; i < nact; i++ {
				v, c := dat.D[k][i], cells[i]
				ys[i][k], as[i][k] = c.ComputeDaily(v[x["Rainfall"]], v[x["Snowfall"]], v[x["MinDailyT"]], v[x["MaxDailyT"]], patm, dt)
			}
		}
		if err := saveGOB(metOutFP+"frc.y.gob", ys); err != nil {
			log.Fatalf("%v", err)
		}
		if err := saveGOB(metOutFP+"frc.ep.gob", as); err != nil {
			log.Fatalf("%v", err)
		}
	} else {
		// build .met
		hnew := hdr.Copy()
		hnew.SetWBDC(met.AtmosphericYield + met.AtmosphericDemand)
		mw, err := met.NewWriter(metOutFP, hnew)
		defer mw.Close()
		if err != nil {
			log.Fatalf("%v", err)
		}
		for k, dt := range dat.T {
			fmt.Println(dt)
			a, j := make([]float32, nact*2), 0
			for i := 0; i < nact; i++ {
				v, c := dat.D[k][i], cells[i]
				y, ep := c.ComputeDaily(v[x["Rainfall"]], v[x["Snowfall"]], v[x["MinDailyT"]], v[x["MaxDailyT"]], patm, dt)
				a[j] = float32(y)
				a[j+1] = float32(ep)
				j += 2
			}
			if err := mw.Add(a); err != nil {
				log.Fatalf("%v", err)
			}
		}
	}
}

func saveIntersect(fp string, gd *grid.Definition) {
	mdlgd, err := grid.ReadGDEF(gdefMdlFP, false)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if mdlgd.Cw > gd.Cw {
		log.Fatalf("saveIntersect is intended for met grids at a lower resultion than the model grid")
	}
	intsct := mdlgd.Intersect(gd)

	a, x := make(map[int]int, len(mdlgd.Sactives)), make(map[int]int, len(gd.Sactives))
	for i, c := range gd.Sactives {
		x[c] = i
	}
	for _, c := range mdlgd.Sactives {
		if _, ok := intsct[c]; !ok {
			log.Fatalf("saveIntersect error: cell %d not found", c)
		}
		a[c] = x[intsct[c][0]]
	}

	// save as gob
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		log.Fatalf("%v", err)
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(a)
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func saveGOB(fp string, d [][]float64) error {
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		return err
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(d)
	if err != nil {
		return err
	}
	return nil
}

func loadGOB(fp string) ([][]float64, error) {
	var d [][]float64
	f, err := os.Open(fp)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&d)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func check() {
	fmt.Println(" ..checking gob..")

	p := func(fp string) {
		fmt.Printf("  loading: %s\n", fp)
		gob, err := loadGOB(fp)
		if err != nil {
			log.Fatalf("%v", err)
		}

		n, nstep := 3, len(gob[0])
		rand.Seed(time.Now().UnixNano())
		x, ys := make([]float64, nstep), make(map[string][]float64, n)
		nm := mmio.FileName(fp, true)
		for i := 0; i < n; i++ {
			ii := rand.Int31n(int32(len(gob)))
			sii := fmt.Sprintf("%d", ii)
			ys[sii] = make([]float64, nstep)
			ys[sii] = gob[ii]
			if i == 0 {
				lgy := make([]float64, 0)
				for j := 0; j < nstep; j++ {
					x[j] = float64(j)
					if gob[ii][j] > 0. {
						lgy = append(lgy, math.Log10(gob[ii][j]))
					}
				}
				mmio.Histo(nm+".LgHist.png", lgy, 30)
				mmio.HistoGT0(nm+".hist.png", gob[ii], 30)
			}
		}
		mmio.Line(nm+".hyd.png", x, ys, 48.)
	}

	p(metOutFP + "frc.y.gob")
	p(metOutFP + "frc.ep.gob")

	// func() {
	// 	fmt.Println("  checking intersection")
	// 	var a []int
	// 	f, err := os.Open(metOutFP + "metIntersect.gob")
	// 	defer f.Close()
	// 	if err != nil {
	// 		log.Fatalf("%v", err)
	// 	}
	// 	enc := gob.NewDecoder(f)
	// 	err = enc.Decode(&a)
	// 	if err != nil {
	// 		log.Fatalf("%v", err)
	// 	}

	// 	mdlgd, err := grid.ReadGDEF(gdefMdlFP, false)
	// 	if err != nil {
	// 		log.Fatalf("%v", err)
	// 	}
	// 	m := make(map[int]int, mdlgd.Nactives())
	// 	for i, c := range mdlgd.Sactives() {
	// 		m[c] = a[i]
	// 	}
	// 	mmio.WriteIMAP(metOutFP+"metIntersect.gob.test.imap", m)
	// }()

	os.Exit(0)
}

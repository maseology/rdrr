package main

import (
	"fmt"
	"log"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/goHydro/met"
	"github.com/maseology/goHydro/tem"
	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/prep"
)

const (
	gdefFP = "M:/ORMGP/met/ORMGP_500a.gdef"
	demFP  = "M:/ORMGP/met/ORMGP_500.hdem"
	metFP  = "M:/ORMGP/met/ORMGP_500a_YCDB.met"

	metOutFP = "M:/ORMGP/met/RDRR_500a_YCDB.met"
)

func main() {
	tt := mmio.NewTimer()
	defer tt.Print("prep complete!")

	gd, err := grid.ReadGDEF(gdefFP, true)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if gd.Nactives() <= 0 {
		log.Fatalf("error: grid definition requires active cells")
	}

	tem, err := tem.NewTEM(demFP)
	if err != nil {
		log.Fatalf("%v", err)
	}
	for _, i := range gd.Actives() {
		if tem.TEC[i].Z == -9999. {
			log.Fatalf("no elevation assigned to cell %d", i)
		}
	}

	hdr, dat, err := met.ReadBigMET(metFP, true)
	if err != nil {
		log.Fatalf("%v", err)
	}

	// build
	hnew := hdr.Copy()
	hnew.SetWBDC(met.AtmosphericYield + met.AtmosphericDemand)
	mw, err := met.NewWriter(metOutFP, hnew)
	defer mw.Close()
	if err != nil {
		log.Fatalf("%v", err)
	}
	cells := make([]prep.Cell, gd.Nactives())
	for k, dt := range dat.T {
		fmt.Println(dt)
		a, j := make([]float32, gd.Nactives()*2), 0
		for i := 0; i < gd.Nactives(); i++ {
			v, c := dat.D[k][i], cells[i]
			y, ep := c.ComputeDaily(v[0], v[1], v[2], v[3], dt.YearDay())
			// y, ep := c.ComputeDaily(v[met.AtmosphericDemand], v[met.AtmosphericDemand], v[met.AtmosphericDemand], v[met.AtmosphericDemand], 1)
			fmt.Println(y, ep)
			a[j] = float32(y)
			a[j+1] = float32(ep)
			j += 2
		}
		if err := mw.Add(a); err != nil {
			log.Fatalf("%v", err)
		}
	}

	fmt.Println("\n", gd.Nactives(), tem.NumCells(), hdr.Nloc(), hdr.Nstep(), len(dat.T))
}

package model

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/maseology/mmio"
	"github.com/maseology/montecarlo/smpln"
	mrg63k3a "github.com/maseology/pnrg/MRG63k3a"
)

// SampleSurfGeo samples like SampleDefault, but adds sampling of surficial geology types to an outlet cellID (=-1 for full-domain model)
func (d *Domain) SampleSurfGeo(outdir string, nsmpl, outlet int) {
	fmt.Println("Building Sub Domain..")
	var b subdomain
	if d.Frc == nil {
		log.Fatalf(" Daomain.SampleSurfGeo error: no forcings made available\n")
	}

	b = d.newSubDomain(d.Frc, outlet)
	fmt.Printf(" catchment area: %.1f km² (%s cells)\n", b.contarea/1000./1000., mmio.Thousands(int64(b.ncid)))

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())

	tt := mmio.NewTimer()
	fmt.Printf(" number of subwatersheds: %d\n", len(b.rtr.SwsCidXR))
	fmt.Printf(" running %d samples from %d dimensions..\n", nsmpl, nSGeoSmplDim)

	printParams := func(m, grdMin, kstrm, mcasc, soildepth, dinc float64, ksat []float64, mdir string) {
		tw, err := mmio.NewTXTwriter(mdir + "params.txt")
		defer tw.Close()
		if err != nil {
			log.Fatalf("SampleDefault.printParams error: %v", err)
		}
		tw.WriteLine(mmio.MMtime(time.Now()))
		tw.WriteLine(mdir)
		tw.WriteLine(fmt.Sprintf("m\t%f", m))
		tw.WriteLine(fmt.Sprintf("grdMin\t%f", grdMin))
		tw.WriteLine(fmt.Sprintf("kstrm\t%f", kstrm))
		tw.WriteLine(fmt.Sprintf("mcasc\t%f", mcasc))
		tw.WriteLine(fmt.Sprintf("soildepth\t%f", soildepth))
		tw.WriteLine(fmt.Sprintf("ksat\t%e", ksat))
		tw.WriteLine(fmt.Sprintf("dinc\t%f", dinc))
	}

	gen := func(u []float64) float64 {
		mdir := newMCdir()
		m, gdn, kstrm, mcasc, soildepth, dinc, ksat := parSurfGeo(u)
		go printParams(m, gdn, kstrm, mcasc, soildepth, dinc, ksat, mdir)
		smpl := b.surfgeoSample(m, gdn, kstrm, mcasc, soildepth, ksat)
		smpl.dir = mdir
		of := b.evaluate(&smpl, dinc, m, false)
		WaitMonitors()
		compressMC2(mdir)
		fmt.Print(".")
		return of
	}
	sp := smpln.NewLHC(rng, nsmpl, nSGeoSmplDim, false)
	for k := 0; k < nsmpl; k++ {
		ut := make([]float64, nSGeoSmplDim)
		for j := 0; j < nSGeoSmplDim; j++ {
			ut[j] = sp.U[j][k]
		}
		gen(ut)
	}
	tt.Lap("results save complete")
}

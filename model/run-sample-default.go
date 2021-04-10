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

// SampleDefault samples a default/parsimonus-parameter to an outlet cellID (=-1 for full-domain model)
func (d *Domain) SampleDefault(outdir string, nsmpl, outlet int) {
	fmt.Println("Building Sub Domain..")
	var b subdomain
	if d.Frc == nil {
		log.Fatalf(" basin.RunMaster error: no forcings made available\n")
	}

	b = d.newSubDomain(d.Frc, outlet)
	fmt.Printf(" catchment area: %.1f km² (%s cells)\n", b.contarea/1000./1000., mmio.Thousands(int64(b.ncid)))

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())

	tt := mmio.NewTimer()
	fmt.Printf(" number of subwatersheds: %d\n", len(b.rtr.SwsCidXR))
	fmt.Printf(" running %d samples from %d dimensions..\n", nsmpl, nDefltSmplDim)

	printParams := func(m, grdMin, kstrm, mcasc, soildepth, kfact, dinc float64, mdir string) {
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
		tw.WriteLine(fmt.Sprintf("kfact\t%f", kfact))
		tw.WriteLine(fmt.Sprintf("dinc\t%f", dinc))
	}

	modelIsSmall := false
	if modelIsSmall {
		// gen := func(u []float64) {
		// 	setMCdir()
		// 	// m, hmax, smax, dinc, soildepth, kfact := par6(u)
		// 	m, grng, soildepth, kfact := par4(u)
		// 	go printParams(m, grng, soildepth, kfact)
		// 	smpl := b.toDefaultSample(m, grng, soildepth, kfact)
		// 	b.evaluate(&smpl, 0., m, false)
		// 	WaitMonitors()
		// 	compressMC()
		// }

		// montecarlo.GenerateSamples(gen, nSmplDim, nsmpl)
		// tt.Lap("\nsampling complete")

		// u, f, d := montecarlo.RankedUnBiased(gen, nSmplDim, nsmpl)

		// tt.Lap("\nsampling complete")
		// t, err := mmio.NewTXTwriter(mcdir + "MCsummary.csv")
		// if err != nil {
		// 	log.Fatalf("SampleDefault %s save error: %v", mcdir+"MCsummary.csv", err)
		// }
		// t.WriteLine(fmt.Sprintf("rank(of %d),eval,m,smax,dinc,soildepth,kfact", nsmpl))
		// for i, dd := range d {
		// 	m, grng, soildepth, kfact := par4(u[dd])
		// 	t.WriteLine(fmt.Sprintf("%d,%f,%f,%f,%f,%f", i+1, 1.-f[dd], m, grng, soildepth, kfact))
		// }
		// t.Close()
	} else {
		gen := func(u []float64) float64 {
			mdir := newMCdir()
			m, gdn, kstrm, mcasc, soildepth, kfact, dinc := par7(u)
			go printParams(m, gdn, kstrm, mcasc, soildepth, kfact, dinc, mdir)
			smpl := b.defaultSample(m, gdn, kstrm, mcasc, soildepth, kfact)
			smpl.dir = mdir
			of := b.evaluate(&smpl, dinc, m, false, evalMC)
			WaitMonitors()
			compressMC2(mdir)
			fmt.Print(".")
			return of
		}
		sp := smpln.NewLHC(rng, nsmpl, nDefltSmplDim, false)
		for k := 0; k < nsmpl; k++ {
			ut := make([]float64, nDefltSmplDim)
			for j := 0; j < nDefltSmplDim; j++ {
				ut[j] = sp.U[j][k]
			}
			gen(ut)
			fmt.Print(".")
		}
	}
	tt.Lap("results save complete")
}

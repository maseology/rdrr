package model

import (
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"sync"
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
		log.Fatalf(" Domain.SampleSurfGeo error: no forcings made available\n")
	}

	b = d.newSubDomain(d.Frc, outlet)
	fmt.Printf(" catchment area: %.1f km² (%s cells)\n", b.contarea/1000./1000., mmio.Thousands(int64(b.ncid)))

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())

	tt := mmio.NewTimer()
	fmt.Printf(" number of subwatersheds: %d\n", len(b.rtr.SwsCidXR))
	fmt.Printf(" running %d samples from %d dimensions..\n", nsmpl, nSGeoSmplDim)

	var mu sync.Mutex
	printParams := func(m, grdMin, kstrm, mcasc, urbDiv, soildepth, dinc float64, ksat []float64, mdir string) {
		mu.Lock()
		defer mu.Unlock()
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
		tw.WriteLine(fmt.Sprintf("urbDiv\t%f", urbDiv))
		tw.WriteLine(fmt.Sprintf("soildepth\t%f", soildepth))
		tw.WriteLine(fmt.Sprintf("dinc\t%f", dinc))
		tw.WriteLine(fmt.Sprintf("ksat\t%e", ksat))
	}

	gen := func(u []float64) float64 {
		mdir := newMCdir()
		m, gdn, kstrm, mcasc, urbDiv, soildepth, dinc, ksat := parSurfGeo(u)
		go printParams(m, gdn, kstrm, mcasc, urbDiv, soildepth, dinc, ksat, mdir)
		smpl := b.surfgeoSample(m, gdn, kstrm, mcasc, urbDiv, soildepth, ksat)
		smpl.dir = mdir
		of := b.evaluate(&smpl, dinc, m, false)
		WaitMonitors()
		compressMC2(mdir)
		fmt.Print(".")
		return of
	}
	sp := smpln.NewLHC(rng, nsmpl, nSGeoSmplDim, false)

	nPara := runtime.GOMAXPROCS(0) / 2
	if nPara > 1 { // ONLY USE FOR SMALLER MODELS !!!
		var wg sync.WaitGroup
		k := 0
		for k < nsmpl {
			for t := 0; t < nPara; t++ {
				if k < nsmpl {
					wg.Add(1)
					go func(k int) {
						ut := make([]float64, nSGeoSmplDim)
						for j := 0; j < nSGeoSmplDim; j++ {
							ut[j] = sp.U[j][k]
						}
						gen(ut)
						wg.Done()
					}(k)
				}
				k++
			}
			wg.Wait()
		}
	} else { // serial
		for k := 0; k < nsmpl; k++ {
			ut := make([]float64, nSGeoSmplDim)
			for j := 0; j < nSGeoSmplDim; j++ {
				ut[j] = sp.U[j][k]
			}
			gen(ut)
		}
	}
	tt.Lap("results save complete")
}

package rdrr

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/maseology/mmio"
	"github.com/maseology/montecarlo/smpln"
	mrg63k3a "github.com/maseology/pnrg/MRG63k3a"
)

func GenerateSamples(gen func(u []float64) Evaluator, frc *Forcing, nc, n, p, nwrkrs int, outdir string) {

	// set up workers
	done := make(chan interface{})
	rin := make(chan realization, nwrkrs)
	defer close(done)
	rout := evalstream(done, rin, nwrkrs)

	// build sampling plan
	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	sp := smpln.NewLHC(rng, n, p, false) // smpln.NewHalton(s, n)

	outdirbatch := outdir + time.Now().Format("060102150405") // batch number = date
	func() {                                                  // save sample space
		lns := make([]string, n)
		for k := 0; k < n; k++ {
			lns[k] = fmt.Sprint(k)
			for j := 0; j < p; j++ {
				lns[k] += fmt.Sprintf(",%f", sp.U[j][k])
			}
		}
		mmio.WriteLines(outdirbatch+".samplespace.csv", lns)
	}()

	for k := 0; k < n; k++ {
		fmt.Printf(" >> releasing sample %d", k+1)
		go func(k int, outdirprfx string) {
			ut := make([]float64, p)
			for j := 0; j < p; j++ {
				ut[j] = sp.U[j][k]
			}

			ev := gen(ut) // generate realization
			ev.evaluate(rin, rout, frc, nc, outdirprfx)

		}(k, fmt.Sprintf("%s.%d.", outdirbatch, k))

		// breaking to stagger runs
		func() {
			time.Sleep(time.Second * 5)
			for {
				if len(rin) <= nwrkrs/2 {
					return
				}
			}
		}()
	}
}

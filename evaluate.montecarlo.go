package rdrr

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/maseology/mmio"
	"github.com/maseology/montecarlo/smpln"
	mrg63k3a "github.com/maseology/pnrg/MRG63k3a"
)

func GenerateSamples(gen func(u []float64) Evaluator, frc *Forcing, nc, n, p, nwrkrs int, outdir string) {
	var wg sync.WaitGroup

	// build sampling plan
	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	sp := smpln.NewLHC(rng, n, p, false) // smpln.NewHalton(s, n)

	outdirbatch := outdir + time.Now().Format("060102150405") // batch number = date
	println("running batch " + outdirbatch)
	func() { // save sample space
		lns := make([]string, n)
		for k := 0; k < n; k++ {
			lns[k] = fmt.Sprint(k)
			for j := 0; j < p; j++ {
				lns[k] += fmt.Sprintf(",%f", sp.U[j][k])
			}
		}
		mmio.WriteLines(outdirbatch+".samplespace.csv", lns)
	}()

	wg.Add(n)
	// prcd := make(chan bool)
	for k := 0; k < n; k++ {
		fmt.Printf(" >> releasing sample %d\n", k+1)
		go func(k int, outdirprfx string) {
			ut := make([]float64, p)
			for j := 0; j < p; j++ {
				ut[j] = sp.U[j][k]
			}

			ev := gen(ut) // generate realization
			ev.evaluate(frc, nc, nwrkrs, outdirprfx, nil)
			fmt.Printf(" >>> complete sample %d\n", k+1)
			wg.Done()
		}(k, fmt.Sprintf("%s.%05d.", outdirbatch, k))

		time.Sleep(time.Minute * 5) ////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
		// <-prcd
	}
	wg.Wait()
	// close(prcd)
}

// // breaking to stagger runs
// func stagger(done <-chan interface{}, rin chan realization, t int) <-chan bool {
// 	prcd := make(chan bool)
// 	go func() {
// 		defer close(prcd)
// 	// loop:
// 	// 	for {
// 	// 		select {
// 	// 		case <-done:
// 	// 			return
// 	// 		default:
// 	// 			if len(rin) >= t {
// 	// 				break loop
// 	// 			}
// 	// 		}
// 	// 	}

// 		for {
// 			select {
// 			case <-done:
// 				return
// 			default:
// 				if len(rin) <= t {
// 					prcd <- true
// 				}
// 			}
// 		}
// 	}()
// 	return prcd
// }

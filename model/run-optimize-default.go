package model

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/maseology/glbopt"
	mrg63k3a "github.com/maseology/pnrg/MRG63k3a"
)

// OptimizeDefault solves a default-parameter model to a given basin outlet
func (d *Domain) OptimizeDefault(frc *FORC, outlet int) (float64, []float64) {
	var b subdomain
	if frc == nil {
		if d.Frc == nil {
			log.Fatalf(" basin.RunDefault error: no forcings made available\n")
		}
		b = d.newSubDomain(d.Frc, outlet) // gauge outlet cell id found in .met file
	} else {
		log.Fatalf(" to fix")
		// b = masterDomain.newSubDomain(loadForcing(metfp, true)) // gauge outlet cell id found in .met file
	}

	fmt.Printf(" catchment area: %.1f km²\n", b.contarea/1000./1000.)
	fmt.Printf(" building sample HRUs and TOPMODEL\n")
	b.print()

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())

	gen := func(u []float64) float64 {
		m, _, kstrm, mcasc, soildepth, kfact, _ := par7(u)
		smpl := b.defaultSample(m, kstrm, mcasc, soildepth, kfact)
		return b.evaluate(&smpl, false, eval)
	}

	fmt.Println(" optimizing..")
	uFinal, _ := glbopt.SCE(32, nDefltSmplDim, rng, gen, true) //runtime.GOMAXPROCS(0) //////////////////////////////////////////////////////////////////////////
	// uFinal, _ := glbopt.SurrogateRBF(500, nDefltSmplDim, rng, gen)

	m, _, kstrm, mcasc, soildepth, kfact, _ := par7(uFinal)
	fmt.Printf("\nfinal parameters:\n\tTMQm:=\t\t%v\n\tkstrm:=\t\t%v\n\tmcasc:=\t\t%v\n\tsoildepth:=\t%v\n\tkfact:=\t\t%v\n\n", m, kstrm, mcasc, soildepth, kfact)
	final := b.defaultSample(m, kstrm, mcasc, soildepth, kfact)
	return b.evaluate(&final, true, evalWB), []float64{m, kstrm, mcasc, soildepth, kfact}
}

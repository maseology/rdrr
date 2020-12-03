package model

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/maseology/glbopt"
	mrg63k3a "github.com/maseology/pnrg/MRG63k3a"
)

// // Optimize solves the model to a give basin outlet
// func Optimize(ldr *Loader) {
// 	if masterDomain.IsEmpty() {
// 		masterDomain = newDomain(ldr)
// 	}
// 	if masterDomain.frc.h.Nloc() != 1 && masterDomain.frc.h.LocationCode() <= 0 {
// 		log.Fatalf(" basin.RunDefault error: unrecognized .met type\n")
// 	}
// 	b := masterDomain.newSubDomain(int(masterDomain.frc.h.Locations[0][0].(int32))) // gauge outlet id found in .met file

// 	nsmpl := len(b.mpr.lu) + len(b.mpr.sg)*3 + 4

// 	rng := rand.New(mrg63k3a.New())
// 	rng.Seed(time.Now().UnixNano())
// 	ver := b.evalCascWB

// 	gen := func(u []float64) float64 {
// 		smpl := b.toSampleU(u...)
// 		return ver(&smpl, 0., false)
// 	}

// 	fmt.Println(" optimizing..")
// 	// uFinal, _ := glbopt.SCE(runtime.GOMAXPROCS(0), nsmpl, rng, gen, true)
// 	uFinal, _ := glbopt.SurrogateRBF(500, nsmpl, rng, gen)

// 	fmt.Printf("\nfinal parameters: %v\n", uFinal)
// 	final := b.toSampleU(uFinal...)
// 	ver(&final, 0., true)
// }

// OptimizeDefault solves a default-parameter model to a given basin outlet
func OptimizeDefault(frc *FORC, outlet int) (float64, []float64) {
	if masterDomain.IsEmpty() {
		log.Fatalf(" basin.RunDefault error: masterDomain is empty")
	}
	var b subdomain
	if frc == nil {
		if masterDomain.frc == nil {
			log.Fatalf(" basin.RunDefault error: no forcings made available\n")
		}
		b = masterDomain.newSubDomain(masterDomain.frc, outlet) // gauge outlet cell id found in .met file
	} else {
		log.Fatalf(" to fix")
		// b = masterDomain.newSubDomain(loadForcing(metfp, true)) // gauge outlet cell id found in .met file
	}

	fmt.Printf(" catchment area: %.1f km²\n", b.contarea/1000./1000.)
	fmt.Printf(" building sample HRUs and TOPMODEL\n")
	b.print()
	// return 0., []float64{0.}

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())

	gen := func(u []float64) float64 {
		// m, hmax, smax, dinc, soildepth, kfact := par6(u)
		m, grng, soildepth, kfact := par4(u)
		smpl := b.toDefaultSample(m, grng, soildepth, kfact)
		return 1. - b.evaluate(&smpl, 0., m, false)
	}

	fmt.Println(" optimizing..")
	uFinal, _ := glbopt.SCE(1, nSmplDim, rng, gen, true) //runtime.GOMAXPROCS(0) //////////////////////////////////////////////////////////////////////////
	// uFinal, _ := glbopt.SurrogateRBF(500, nSmplDim, rng, gen)

	// m, hmax, smax, dinc, soildepth, kfact := par6(uFinal)
	// fmt.Printf("\nfinal parameters:\n\tTMQm:=\t\t%v\n\thmax:=\t\t%v\n\tsmax:=\t\t%v\n\tdinc:=\t\t%v\n\tsoildepth:=\t%v\n\tkfact:=\t\t%v\n\n", m, hmax, smax, dinc, soildepth, kfact)
	m, grng, soildepth, kfact := par4(uFinal)
	fmt.Printf("\nfinal parameters:\n\tTMQm:=\t\t%v\n\tgrng:=\t\t%v\n\tsoildepth:=\t%v\n\tkfact:=\t\t%v\n\n", m, grng, soildepth, kfact)
	final := b.toDefaultSample(m, grng, soildepth, kfact)
	return b.evaluate(&final, 0., m, true), []float64{m, grng, 0., soildepth, kfact}
}

// // OptimizeDefault1 solves a default-parameter model to a given basin outlet
// // changes only 1 basin-wide parameter (choice hard-coded)
// func OptimizeDefault1(metfp string) (float64, []float64) {
// 	if masterDomain.IsEmpty() {
// 		log.Fatalf(" basin.RunDefault error: masterDomain is empty")
// 	}
// 	var b subdomain
// 	if len(metfp) == 0 {
// 		if masterDomain.frc == nil {
// 			log.Fatalf(" basin.RunDefault error: no forcings made available\n")
// 		}
// 		b = masterDomain.newSubDomain(masterDomain.frc, -1) // gauge outlet cell id found in .met file
// 	} else {
// 		log.Fatalf(" to fix...")
// 		// b = masterDomain.newSubDomain(loadForcing(metfp, true)) // gauge outlet cell id found in .met file
// 	}
// 	dt, y, ep, obs, intvl, nstep := b.getForcings()

// 	fmt.Printf(" catchment area: %.1f km²\n", b.contarea/1000./1000.)
// 	fmt.Printf(" building sample HRUs and TOPMODEL\n\n")

// 	const (
// 		TMQm      = 0.004191296639278929
// 		smax      = 0.2336020076838129
// 		dinc      = 1.
// 		hmax      = .01
// 		soildepth = .1
// 		kfact     = 1.
// 	)

// 	smpl1 := b.toDefaultSample(TMQm, smax, soildepth, kfact)
// 	par1 := func(u []float64) float64 {
// 		// m := mmaths.LogLinearTransform(0.001, 1., u[0])
// 		// smax := mmaths.LogLinearTransform(0.1, 10., u[0])
// 		soildepth := mmaths.LinearTransform(-1., 1., u[0])
// 		return soildepth
// 	}
// 	gen := func(u []float64) float64 {
// 		smpl := smpl1.copy() // b.toDefaultSample(TMQm, fcasc)
// 		return b.eval(&smpl, dt, y, ep, obs, intvl, nstep, dinc, hmax, par1(u), false)
// 	}

// 	fmt.Println(" optimizing..")
// 	uFinal, _ := glbopt.Fibonacci(gen)

// 	sldpth := par1([]float64{uFinal})
// 	fmt.Printf("\nfinal parameters:\n\tTMQm:\t\t%v\n\tsmax:\t\t%v\n\tdinc:\t\t%v\n\tsoildepth:\t%v\n\tkfact:\t\t%v\n\n", TMQm, smax, dinc, sldpth, kfact)
// 	final := smpl1.copy() // b.toDefaultSample(TMQm, fcasc)
// 	return b.eval(&final, dt, y, ep, obs, intvl, nstep, dinc, hmax, sldpth, true), []float64{TMQm, smax, sldpth, kfact}
// }

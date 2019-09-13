package basin

import (
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/maseology/glbopt"
	"github.com/maseology/mmaths"
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
// changes only 3 basin-wide parameters (Qo, topm, fcasc); freeboard set to 0.
func OptimizeDefault(metfp string) (float64, []float64) {
	if masterDomain.IsEmpty() {
		log.Fatalf(" basin.RunDefault error: masterDomain is empty")
	}
	var b subdomain
	if len(metfp) == 0 {
		if masterDomain.frc == nil {
			log.Fatalf(" basin.RunDefault error: no forcings made available\n")
		}
		b = masterDomain.newSubDomain(masterForcing()) // gauge outlet cell id found in .met file
	} else {
		b = masterDomain.newSubDomain(loadForcing(metfp, true)) // gauge outlet cell id found in .met file
	}

	fmt.Printf(" catchment area: %.1f km²\n", b.contarea/1000./1000.)
	fmt.Printf(" building sample HRUs and TOPMODEL\n\n")

	ndim := 3 // defaulting freeboard=0.

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	ver := b.evalCascWB

	par4 := func(u []float64) (m, fcasc, Qo, freeboard float64) {
		m = mmaths.LogLinearTransform(0.01, 10., u[0])
		fcasc = mmaths.LogLinearTransform(0.001, 10., u[1])
		Qo = mmaths.LinearTransform(0., 1., u[2])
		freeboard = 0. // mmaths.LinearTransform(-1., 1., u[3])
		return
	}
	gen := func(u []float64) float64 {
		m, fcasc, Qo, freeboard := par4(u)
		smpl := b.toDefaultSample(m, fcasc)
		// Qo *= b.frc.h.IntervalSec() / 1000. / 365.24 / 86400. // [mm/yr] to [m/ts]
		return ver(&smpl, Qo, freeboard, false)
	}

	fmt.Println(" optimizing..")
	uFinal, _ := glbopt.SCE(runtime.GOMAXPROCS(0), ndim, rng, gen, true)
	// uFinal, _ := glbopt.SurrogateRBF(500, ndim, rng, gen)

	m, fcasc, Qo, freeboard := par4(uFinal)
	fmt.Printf("\nfinal parameters:\n\tTMQm:\t%v\n\tfcasc:\t%v\n\tQo:\t%v\n\tfrebrd:\t%v\n\n", m, fcasc, Qo, freeboard)
	final := b.toDefaultSample(m, fcasc)
	return ver(&final, Qo, freeboard, true), []float64{m, fcasc, Qo, freeboard}
}

// OptimizeDefault1 solves a default-parameter model to a given basin outlet
// changes only 1 basin-wide parameter (choice hard-coded)
func OptimizeDefault1(metfp string) (float64, []float64) {
	if masterDomain.IsEmpty() {
		log.Fatalf(" basin.RunDefault error: masterDomain is empty")
	}
	var b subdomain
	if len(metfp) == 0 {
		if masterDomain.frc == nil {
			log.Fatalf(" basin.RunDefault error: no forcings made available\n")
		}
		b = masterDomain.newSubDomain(masterForcing()) // gauge outlet cell id found in .met file
	} else {
		b = masterDomain.newSubDomain(loadForcing(metfp, true)) // gauge outlet cell id found in .met file
	}

	fmt.Printf(" catchment area: %.1f km²\n", b.contarea/1000./1000.)
	fmt.Printf(" building sample HRUs and TOPMODEL\n\n")

	ver := b.evalCascWB

	const (
		TMQm  = 0.004191296639278929
		fcasc = 0.2336020076838129
		// freeboard = 0.
	)

	smpl1 := b.toDefaultSample(TMQm, fcasc)
	par1 := func(u []float64) float64 {
		// m := mmaths.LogLinearTransform(0.001, 1., u[0])
		// fcasc := mmaths.LogLinearTransform(0.1, 10., u[0])
		freeboard := mmaths.LinearTransform(-1., 1., u[0])
		return freeboard
	}
	gen := func(u []float64) float64 {
		smpl := smpl1.copy() // b.toDefaultSample(TMQm, fcasc)
		return ver(&smpl, 1., par1(u), false)
	}

	fmt.Println(" optimizing..")
	uFinal, _ := glbopt.Fibonacci(gen)

	freeboard := par1([]float64{uFinal})
	fmt.Printf("\nfinal parameters:\n\tTMQm:\t%v\n\tfcasc:\t%v\n\tfrebrd:\t%v\n\n", TMQm, fcasc, freeboard)
	final := smpl1.copy() // b.toDefaultSample(TMQm, fcasc)
	return ver(&final, 1., freeboard, true), []float64{TMQm, fcasc, freeboard}

	// p0, p1, p2 := par3(uFinal)
	// fmt.Printf("\nfinal parameters:\n\tQo:\t%v\n\tTMQm:\t%v\n\tfcasc:\t%v\n\n", p0, p1, p2)
	// final := b.toDefaultSample(p0, p1, p2)
	// return ver(&final, 0., true), []float64{p0, p1, p2}
}

// func OptimizeDefault1(ldr *Loader, Qomm, m, fcasc float64) (float64, []float64) {
// 	if masterDomain.IsEmpty() {
// 		masterDomain = newDomain(ldr)
// 	}
// 	if masterDomain.frc.h.Nloc() != 1 && masterDomain.frc.h.LocationCode() <= 0 {
// 		log.Fatalf(" basin.RunDefault error: unrecognized .met type\n")
// 	}
// 	b := masterDomain.newSubDomain(int(masterDomain.frc.h.Locations[0][0].(int32))) // gauge outlet id found in .met file

// 	fmt.Printf(" catchment area: %.1f km²\n", b.contarea/1000./1000.)
// 	fmt.Printf(" building sample HRUs and TOPMODEL\n\n")

// 	ver := b.evalCascWB

// 	par1 := func(u []float64) float64 {
// 		Qomm = mmaths.LogLinearTransform(0.001, 1., u[0])
// 		// m = mmaths.LogLinearTransform(0.001, 1., u[1])
// 		// fcasc = mmaths.LogLinearTransform(0.1, 10., u[2])
// 		return Qomm
// 	}
// 	gen := func(u []float64) float64 {
// 		smpl := b.toDefaultSample(par1(u), m, fcasc)
// 		return ver(&smpl, 0., false)
// 	}

// 	fmt.Println(" optimizing..")
// 	uFinal, _ := glbopt.Fibonacci(gen)

// 	Qomm = par1([]float64{uFinal})
// 	fmt.Printf("\nfinal parameter:\t%v\n", Qomm)
// 	final := b.toDefaultSample(Qomm, m, fcasc)
// 	return ver(&final, 0., true), []float64{Qomm}
// }

// // OptimizeUniform solves a uniform-parameter model to a given basin outlet
// // changes only 3 basin-wide parameters (m, n)
// func OptimizeUniform(ldr *Loader) {
// 	if masterDomain.IsEmpty() {
// 		masterDomain = newDomain(ldr)
// 	}
// 	if masterDomain.frc.h.Nloc() != 1 && masterDomain.frc.h.LocationCode() <= 0 {
// 		log.Fatalf(" basin.RunDefault error: unrecognized .met type\n")
// 	}
// 	b := masterDomain.newSubDomain(int(masterDomain.frc.h.Locations[0][0].(int32))) // gauge outlet id found in .met file

// 	nsmpl := len(b.mpr.lu) + len(b.mpr.sg)*3 + 2

// 	rng := rand.New(mrg63k3a.New())
// 	rng.Seed(time.Now().UnixNano())
// 	ver := b.evalCascWB

// 	gen := func(u []float64) float64 {
// 		smpl := b.toSampleU(u...)
// 		return ver(&smpl, 0., false)
// 	}

// 	fmt.Println(" optimizing..")
// 	uFinal, _ := glbopt.SCE(runtime.GOMAXPROCS(0), nsmpl, rng, gen, true)
// 	// uFinal, _ := glbopt.SurrogateRBF(500, nsmpl, rng, gen)

// 	fmt.Printf("\nfinal parameters: %v\n", uFinal)
// 	final := b.toSampleU(uFinal...)
// 	ver(&final, 0., true)

// 	// const nsmpl = 2

// 	// // sample ranges
// 	// topm := func(u float64) float64 {
// 	// 	return mmaths.LogLinearTransform(0.001, 10., u)
// 	// }
// 	// mann := func(u float64) float64 {
// 	// 	return mmaths.LogLinearTransform(0.0001, 100., u)
// 	// }

// 	// rng := rand.New(mrg63k3a.New())
// 	// rng.Seed(time.Now().UnixNano())
// 	// ver := b.evalCascKineWB

// 	// gen := func(u []float64) float64 {
// 	// 	topm := topm(u[1]) // topmodel m
// 	// 	mann := mann(u[2]) // manning's n
// 	// 	smpl := b.toDefaultSample(topm, mann)
// 	// 	return ver(&smpl, false)
// 	// }

// 	// fmt.Println(" optimizing..")
// 	// uFinal, _ := glbopt.SCE(runtime.GOMAXPROCS(0), nsmpl, rng, gen, true)

// 	// func() {
// 	// 	topm := topm(uFinal[1]) // topmodel m
// 	// 	mann := mann(uFinal[2]) // manning's n
// 	// 	fmt.Printf("\nfinal parameters: %v\n", []float64{topm, mann})
// 	// 	final := b.toDefaultSample(topm, mann)
// 	// 	ver(&final, true)
// 	// }()
// }

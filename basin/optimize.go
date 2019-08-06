package basin

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"

	"github.com/maseology/glbopt"
	"github.com/maseology/mmaths"
	mrg63k3a "github.com/maseology/pnrg/MRG63k3a"
)

// Optimize solves the model to a give basin outlet
func Optimize(ldr *Loader) {
	d := newDomain(ldr)
	b := d.newSubDomain(ldr.Outlet)

	nsmpl := len(b.mpr.lu) + len(b.mpr.sg)*3 + 4

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	ver := b.evalCascWB

	gen := func(u []float64) float64 {
		smpl := b.toSampleU(u...)
		return ver(&smpl, 0., false)
	}

	fmt.Println(" optimizing..")
	// uFinal, _ := glbopt.SCE(runtime.GOMAXPROCS(0), nsmpl, rng, gen, true)
	uFinal, _ := glbopt.SurrogateRBF(500, nsmpl, rng, gen)

	fmt.Printf("\nfinal parameters: %v\n", uFinal)
	final := b.toSampleU(uFinal...)
	ver(&final, 0., true)
}

// OptimizeDefault solves a default-parameter model to a given basin outlet
// changes only 3 basin-wide parameters (topf, topm, fcasc); freeboard set to 0.
func OptimizeDefault(ldr *Loader) (float64, []float64) {
	d := newDomain(ldr)
	b := d.newSubDomain(ldr.Outlet)
	fmt.Printf(" catchment area: %.1f km²\n", b.contarea/1000./1000.)
	fmt.Printf(" building sample HRUs and TOPMODEL\n\n")

	nsmpl := 3 // defaulting freeboard=0.

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	ver := b.evalCascWB

	par3 := func(u []float64) (Qomm, m, fcasc float64) {
		Qomm = mmaths.LogLinearTransform(0.001, 1., u[0])
		m = mmaths.LogLinearTransform(0.001, 1., u[1])
		fcasc = mmaths.LogLinearTransform(0.1, 10., u[2])
		return
	}
	gen := func(u []float64) float64 {
		smpl := b.toDefaultSample(par3(u))
		return ver(&smpl, 0., false)
	}

	fmt.Println(" optimizing..")
	uFinal, _ := glbopt.SCE(runtime.GOMAXPROCS(0), nsmpl, rng, gen, true)
	// uFinal, _ := glbopt.SurrogateRBF(500, nsmpl, rng, gen)

	p0, p1, p2 := par3(uFinal)
	fmt.Printf("\nfinal parameters:\n\tQomm: %v\n\tTMQm: %v\n\tfcasc: %v\n\n", p0, p1, p2)
	final := b.toDefaultSample(p0, p1, p2)
	return ver(&final, 0., true), []float64{p0, p1, p2}
}

// OptimizeDefault1 solves a default-parameter model to a given basin outlet
// changes only 1 basin-wide parameter (choice hard-coded); freeboard set to 0.
func OptimizeDefault1(ldr *Loader, Qomm, m, fcasc float64) (float64, []float64) {
	d := newDomain(ldr)
	b := d.newSubDomain(ldr.Outlet)
	fmt.Printf(" catchment area: %.1f km²\n", b.contarea/1000./1000.)
	fmt.Printf(" building sample HRUs and TOPMODEL\n\n")

	ver := b.evalCascWB

	par1 := func(u []float64) float64 {
		Qomm = mmaths.LogLinearTransform(0.001, 1., u[0])
		// m = mmaths.LogLinearTransform(0.001, 1., u[1])
		// fcasc = mmaths.LogLinearTransform(0.1, 10., u[2])
		return Qomm
	}
	gen := func(u []float64) float64 {
		smpl := b.toDefaultSample(par1(u), m, fcasc)
		return ver(&smpl, 0., false)
	}

	fmt.Println(" optimizing..")
	uFinal, _ := glbopt.Fibonacci(gen)

	Qomm = par1([]float64{uFinal})
	fmt.Printf("\nfinal parameter:\t%v\n", Qomm)
	final := b.toDefaultSample(Qomm, m, fcasc)
	return ver(&final, 0., true), []float64{Qomm}
}

// OptimizeUniform solves a uniform-parameter model to a given basin outlet
// changes only 3 basin-wide parameters (m, n)
func OptimizeUniform(ldr *Loader) {
	d := newUniformDomain(ldr)
	b := d.newSubDomain(ldr.Outlet)

	nsmpl := len(b.mpr.lu) + len(b.mpr.sg)*3 + 2

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	ver := b.evalCascWB

	gen := func(u []float64) float64 {
		smpl := b.toSampleU(u...)
		return ver(&smpl, 0., false)
	}

	fmt.Println(" optimizing..")
	uFinal, _ := glbopt.SCE(runtime.GOMAXPROCS(0), nsmpl, rng, gen, true)
	// uFinal, _ := glbopt.SurrogateRBF(500, nsmpl, rng, gen)

	fmt.Printf("\nfinal parameters: %v\n", uFinal)
	final := b.toSampleU(uFinal...)
	ver(&final, 0., true)

	// const nsmpl = 2

	// // sample ranges
	// topm := func(u float64) float64 {
	// 	return mmaths.LogLinearTransform(0.001, 10., u)
	// }
	// mann := func(u float64) float64 {
	// 	return mmaths.LogLinearTransform(0.0001, 100., u)
	// }

	// rng := rand.New(mrg63k3a.New())
	// rng.Seed(time.Now().UnixNano())
	// ver := b.evalCascKineWB

	// gen := func(u []float64) float64 {
	// 	topm := topm(u[1]) // topmodel m
	// 	mann := mann(u[2]) // manning's n
	// 	smpl := b.toDefaultSample(topm, mann)
	// 	return ver(&smpl, false)
	// }

	// fmt.Println(" optimizing..")
	// uFinal, _ := glbopt.SCE(runtime.GOMAXPROCS(0), nsmpl, rng, gen, true)

	// func() {
	// 	topm := topm(uFinal[1]) // topmodel m
	// 	mann := mann(uFinal[2]) // manning's n
	// 	fmt.Printf("\nfinal parameters: %v\n", []float64{topm, mann})
	// 	final := b.toDefaultSample(topm, mann)
	// 	ver(&final, true)
	// }()
}

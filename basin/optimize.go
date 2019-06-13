package basin

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/maseology/glbopt"
	mrg63k3a "github.com/maseology/pnrg/MRG63k3a"
)

// Optimize solves the model to a give basin outlet
func Optimize(ldr *Loader) {
	d := newDomain(ldr)
	b := d.newSubDomain(ldr.outlet)

	nsmpl := len(b.mpr.lu) + len(b.mpr.sg)*3 + 4

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	ver := b.evalCascKineWB

	gen := func(u []float64) float64 {
		smpl := b.toSampleU(u...)
		return ver(&smpl, false)
	}

	fmt.Println(" optimizing..")
	// uFinal, _ := glbopt.SCE(runtime.GOMAXPROCS(0), nsmpl, rng, gen, true)
	uFinal, _ := glbopt.SurrogateRBF(500, nsmpl, rng, gen)

	fmt.Printf("\nfinal parameters: %v\n", uFinal)
	final := b.toSampleU(uFinal...)
	ver(&final, true)
}

// OptimizeUniform solves a uniform-parameter model to a given basin outlet
// changes only 3 basin-wide parameters (m, n)
func OptimizeUniform(ldr *Loader) {
	d := newUniformDomain(ldr)
	b := d.newSubDomain(ldr.outlet)

	nsmpl := len(b.mpr.lu) + len(b.mpr.sg)*3 + 4

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	ver := b.evalCascKineWB

	gen := func(u []float64) float64 {
		smpl := b.toSampleU(u...)
		return ver(&smpl, false)
	}

	fmt.Println(" optimizing..")
	// uFinal, _ := glbopt.SCE(runtime.GOMAXPROCS(0), nsmpl, rng, gen, true)
	uFinal, _ := glbopt.SurrogateRBF(500, nsmpl, rng, gen)

	fmt.Printf("\nfinal parameters: %v\n", uFinal)
	final := b.toSampleU(uFinal...)
	ver(&final, true)

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

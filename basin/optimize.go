package basin

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/maseology/glbopt"
	"github.com/maseology/mmaths"
	mrg63k3a "github.com/maseology/pnrg/MRG63k3a"
)

// Optimize solves the model to a give basin outlet
func Optimize(ldr *Loader) {
	d := newDomain(ldr)
	b := d.newSubDomain(ldr.outlet)

	const ncmplx = 16
	nsmpl := len(b.mpr.lu) + len(b.mpr.sg)*3 + 5

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	ver := b.evalCascKineWB

	gen := func(u []float64) float64 {
		smpl := b.toSampleU(u...)
		return ver(&smpl, false)
	}

	fmt.Println(" optimizing..")
	uFinal, _ := glbopt.SCE(ncmplx, nsmpl, rng, gen, true)

	fmt.Printf("\nfinal parameters: %v\n", uFinal)
	final := b.toSampleU(uFinal...)
	ver(&final, true)
}

// Optimize3 solves the model to a given basin outlet
// changes only 3 basin-wide parameters (rill, m, n)
func Optimize3(ldr *Loader) {
	d := newDomain(ldr)
	b := d.newSubDomain(ldr.outlet)

	const ncmplx = 16
	const nsmpl = 3

	// sample ranges
	t0 := func(u float64) float64 {
		return mmaths.LogLinearTransform(0.01, 1., u)
	}
	t1 := func(u float64) float64 {
		return mmaths.LogLinearTransform(0.001, 10., u)
	}
	t2 := func(u float64) float64 {
		return mmaths.LogLinearTransform(0.0001, 100., u)
	}

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	ver := b.evalCascKineWB

	gen := func(u []float64) float64 {
		p0 := t0(u[0]) // rill storage
		p1 := t1(u[1]) // topmodel m
		p2 := t2(u[2]) // manning's n
		smpl := b.toDefaultSample(p0, p1, p2)
		return ver(&smpl, false)
	}

	fmt.Println(" optimizing..")
	uFinal, _ := glbopt.SCE(ncmplx, nsmpl, rng, gen, true)

	p0 := t0(uFinal[0]) // rill storage
	p1 := t1(uFinal[1]) // topmodel m
	p2 := t2(uFinal[2]) // manning's n
	fmt.Printf("\nfinal parameters: %v\n", []float64{p0, p1, p2})
	final := b.toDefaultSample(p0, p1, p2)
	ver(&final, true)
}

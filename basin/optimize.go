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
	frc, mdl := ldr.load(1.)
	mdl.t = mdl.t.SubSet(ldr.outlet)
	cids, ds := mdl.t.DownslopeContributingAreaIDs(ldr.outlet) // mdl.t.ContributingAreaIDs(ldr.outlet)
	ncid := len(cids)
	fncid := float64(ncid)
	b := Basin{
		frc:      &frc,
		mdl:      &mdl,
		cids:     cids,
		ds:       ds,
		ncid:     ncid,
		fncid:    fncid,
		contarea: mdl.a * fncid, // basin contributing area [m²]
	}

	// sample ranges
	t0 := func(u float64) float64 {
		return mmaths.LogLinearTransform(0.001, .1, u)
	}
	t1 := func(u float64) float64 {
		return mmaths.LogLinearTransform(0.001, 10., u)
	}
	t2 := func(u float64) float64 {
		return mmaths.LogLinearTransform(0.0001, 1., u)
	}

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	ver := b.evalCascWB

	gen := func(u []float64) float64 {
		p0 := t0(u[0]) // rill storage
		p1 := t1(u[1]) // topmodel m
		p2 := t2(u[2]) // cascade fraction
		smpl := b.toSample(p0, p1, p2)
		return ver(&smpl, false)
	}
	fmt.Println(" optimizing..")
	uFinal, _ := glbopt.SCE(16, 3, rng, gen, true)

	p0 := t0(uFinal[0]) // rill storage
	p1 := t1(uFinal[1]) // topmodel m
	p2 := t2(uFinal[2]) // cascade fraction
	fmt.Printf("\nfinal parameters: %v\n", []float64{p0, p1})
	final := b.toSample(p0, p1, p2)
	ver(&final, true)
}

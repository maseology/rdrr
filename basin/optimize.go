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
	cids := mdl.t.ContributingAreaIDs(ldr.outlet)
	ncid := len(cids)
	fncid := float64(ncid)
	b := Basin{
		frc:      &frc,
		mdl:      &mdl,
		cids:     cids,
		ncid:     ncid,
		fncid:    fncid,
		contarea: mdl.a * fncid, // basin contributing area [m²]
	}

	t0 := func(u float64) float64 {
		return mmaths.LogLinearTransform(0.001, .1, u)
	}
	t1 := func(u float64) float64 {
		return mmaths.LogLinearTransform(0.001, 10., u)
	}

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())

	gen := func(u []float64) float64 {
		p0 := t0(u[0]) // rill storage
		p1 := t1(u[1]) // topmodel m
		smpl := b.toSample(p0, p1)
		return b.evalWB(&smpl, false)
	}
	fmt.Println(" optimizing..")
	uFinal, _ := glbopt.SCE(16, 2, rng, gen, true)

	p0 := mmaths.LogLinearTransform(0.001, .1, uFinal[0])  // rill storage
	p1 := mmaths.LogLinearTransform(0.001, 10., uFinal[1]) // topmodel m
	fmt.Printf("\nfinal parameters: %v\n", []float64{p0, p1})
	final := b.toSample(p0, p1)
	b.evalWB(&final, true)
}

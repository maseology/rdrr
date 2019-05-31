package basin

import (
	"log"
	"math"
	"sync"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
	"github.com/maseology/mmaths"
)

const (
	twoThirds  = 2. / 3.
	fiveThirds = 5. / 3.
)

type sample struct {
	ws     hru.WtrShd // hru watershed
	gw     gwru.TMQ   // topmodel
	p0, p1 map[int]float64
	rill   float64
}

func (b *subdomain) toSampleU(u ...float64) sample {
	var wg sync.WaitGroup

	ws := make(hru.WtrShd, b.ncid)
	var gw gwru.TMQ

	// transform sample space
	rill := mmaths.LogLinearTransform(0.01, 1., u[0])
	topm := mmaths.LogLinearTransform(0.001, 10., u[1])
	dsoil := mmaths.LinearTransform(0.01, 1., u[2])
	dpsto := mmaths.LogLinearTransform(0.0001, 0.001, u[3])
	itsto := mmaths.LinearTransform(0.001, 0.004, u[4]) // short and tall vegetation interception
	mann := func(u float64) float64 {
		return mmaths.LogLinearTransform(0.0001, 100., u)
	}
	fc := func(u float64) float64 {
		return mmaths.LinearTransform(0.05, 0.4, u)
	}

	// sample surficial geology types
	ksg, nsg, i := 5, 3, 0
	pksat, ppor, pfc := make(map[int]float64, len(b.mpr.sg)), make(map[int]float64, len(b.mpr.sg)), make(map[int]float64, len(b.mpr.sg))
	for k, sg := range b.mpr.sg {
		pksat[k], ppor[k], _ = sg.Sample(u[ksg+nsg*i], u[ksg+nsg*i+1])
		pfc[k] = fc(u[ksg+nsg*i+2])
		i++
	}

	// sample landuse types
	klu, nlu, i := ksg+len(b.mpr.sg)*nsg, 1, 0
	pn, pfimp, pinfct := make(map[int]float64, len(b.mpr.lu)), make(map[int]float64, len(b.mpr.lu)), make(map[int]float64, len(b.mpr.lu))
	for k, lu := range b.mpr.lu {
		pfimp[k] = lu.Fimp
		pinfct[k] = lu.Intfct
		pn[k] = mann(u[klu+nlu*i])
		i++
	}

	n := make(map[int]float64)
	ts := b.frc.h.IntervalSec()
	assignHRUs := func() {
		defer wg.Done()
		ksat := make(map[int]float64)
		var recurs func(int)
		recurs = func(cid int) {
			var hnew hru.HRU
			ksat[cid] = pksat[b.mpr.isg[cid]]
			n[cid] = pn[b.mpr.ilu[cid]]
			drnsto := ppor[b.mpr.isg[cid]] * (1. - pfc[b.mpr.isg[cid]]) * dsoil
			srfsto := ppor[b.mpr.isg[cid]]*pfc[b.mpr.isg[cid]]*dsoil + itsto*pinfct[b.mpr.ilu[cid]]*(1.-pfimp[b.mpr.ilu[cid]]) + dpsto*pfimp[b.mpr.ilu[cid]]
			hnew.Initialize(drnsto, srfsto, pfimp[b.mpr.ilu[cid]], ksat[cid], ts)
			ws[cid] = &hnew
			for _, upcid := range b.strc.t.UpIDs(cid) {
				recurs(upcid)
			}
		}
		recurs(b.cid0)
	}
	buildTopmodel := func() {
		defer wg.Done()
		ksat := make(map[int]float64)
		var recurs func(int)
		recurs = func(cid int) {
			ksat[cid] = pksat[b.mpr.isg[cid]] * ts // [m/ts]
			for _, upcid := range b.strc.t.UpIDs(cid) {
				recurs(upcid)
			}
		}
		recurs(b.cid0)

		if b.frc.Q0 <= 0. {
			log.Fatalf("toDefaultSample.buildTopmodel error, initial flow for TOPMODEL (Q0) is set to %v", b.frc.Q0)
		}
		medQ := b.frc.Q0 * b.strc.a * float64(len(ksat)) // [m/d] to [m³/d]
		gw.New(ksat, b.strc.t, b.strc.w, medQ, 2*medQ, topm)
	}

	wg.Add(2)
	go assignHRUs()
	go buildTopmodel()
	wg.Wait()

	return sample{
		ws:   ws,
		gw:   gw,
		p0:   b.buildC0(n, ts),
		p1:   b.buildC2(n, ts),
		rill: rill,
	}
}

func (b *subdomain) buildC0(n map[int]float64, ts float64) map[int]float64 {
	c0 := make(map[int]float64, len(b.cids))
	for _, cid := range b.cids {
		c := fiveThirds * math.Sqrt(b.strc.t.TEC[cid].S) * ts / b.strc.w / n[cid]
		c0[cid] = c / (1. + c)
	}
	return c0
}

func (b *subdomain) buildC2(n map[int]float64, ts float64) map[int]float64 {
	c2 := make(map[int]float64, len(b.cids))
	for _, cid := range b.cids {
		c := fiveThirds * math.Sqrt(b.strc.t.TEC[cid].S) * ts / b.strc.w / n[cid]
		c2[cid] = 1. / (1. + c)
	}
	return c2
}

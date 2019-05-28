package basin

import (
	"log"
	"math"
	"sync"

	"github.com/maseology/rdrr/lusg"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
	"github.com/maseology/mmaths"
)

const (
	twoThirds  = 2. / 3.
	fiveThirds = 5. / 3.
)

func (b *subdomain) buildFc(f1 float64) map[int]float64 {
	fc := make(map[int]float64, len(b.cids))
	for _, c := range b.cids {
		fc[c] = math.Min(f1*b.strc.t.TEC[c].S, 1.)
	}
	return fc
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

func (b *subdomain) toDefaultSample(rill, m, n float64) sample {
	var wg sync.WaitGroup

	ts := b.frc.h.IntervalSec()
	if ts <= 0. {
		log.Fatalf("toDefaultSample error, timestep (IntervalSec) = %v", ts)
	}
	ws := make(hru.WtrShd, b.ncid)
	var gw gwru.TMQ
	assignHRUs := func() {
		defer wg.Done()
		var recurs func(int)
		recurs = func(cid int) {
			var ll, gg int
			var ok bool
			if ll, ok = b.mpr.ilu[cid]; !ok {
				log.Fatalf("toDefaultSample.assignHRUs error, no LandUse assigned to cell ID %d", cid)
			}
			if gg, ok = b.mpr.isg[cid]; !ok {
				log.Fatalf("toDefaultSample.assignHRUs error, no SurfGeo assigned to cell ID %d", cid)
			}
			var lu lusg.LandUse
			var sg lusg.SurfGeo
			if lu, ok = b.mpr.lu[ll]; !ok {
				log.Fatalf("toDefaultSample.assignHRUs error, no LandUse assigned of type %d", ll)
			}
			if sg, ok = b.mpr.sg[gg]; !ok {
				log.Fatalf("toDefaultSample.assignHRUs error, no SurfGeo assigned to type %d", gg)
			}

			var h hru.HRU
			h.Initialize(lu.DrnSto, lu.SrfSto, lu.Fimp, sg.Ksat, ts)
			ws[cid] = &h
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
			if gg, ok := b.mpr.isg[cid]; ok {
				if sg, ok := b.mpr.sg[gg]; ok {
					ksat[cid] = sg.Ksat * ts // [m/ts]
					for _, upcid := range b.strc.t.UpIDs(cid) {
						recurs(upcid)
					}
				} else {
					log.Fatalf("toDefaultSample.buildTopmodel error, no SurfGeo assigned to type %d", gg)
				}
			} else {
				log.Fatalf("toDefaultSample.buildTopmodel error, no SurfGeo assigned to cell ID %d", cid)
			}
		}
		recurs(b.cid0)

		if b.frc.Q0 <= 0. {
			log.Fatalf("toDefaultSample.buildTopmodel error, initial flow for TOPMODEL (Q0) is set to %v", b.frc.Q0)
		}
		medQ := b.frc.Q0 * b.strc.a * float64(len(ksat)) // [m/d] to [m³/d]
		gw.New(ksat, b.strc.t, b.strc.w, medQ, 2*medQ, m)
	}

	wg.Add(2)
	go assignHRUs()
	go buildTopmodel()
	wg.Wait()

	na := make(map[int]float64, b.ncid)
	for i := range ws {
		na[i] = n
	}
	return sample{
		ws:   ws,
		gw:   gw,
		p0:   b.buildC0(na, ts), // b.buildFc(f1),
		p1:   b.buildC2(na, ts),
		rill: rill,
	}
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
	mann := func(u float64) float64 {
		return mmaths.LogLinearTransform(0.0001, 100., u)
	}
	fc := func(u float64) float64 {
		return mmaths.LinearTransform(0.05, 0.4, u)
	}
	itcp := func(u float64) float64 { // short and tall vegetation interception
		return mmaths.LinearTransform(0.001, 0.004, u)
	}

	// sample surficial geology types
	ksg, nsg := 4, 3
	pksat, ppor, pfc := make(map[int]float64, len(b.mpr.sg)), make(map[int]float64, len(b.mpr.sg)), make(map[int]float64, len(b.mpr.sg))
	for k, sg := range b.mpr.sg {
		i := -1 /////////////////////////////////////////////
		pksat[i], ppor[i], _ = sg.Sample(u[ksg+nsg*k], u[ksg+nsg*k+1])
		pfc[i] = fc(u[ksg+nsg*k+2])
	}

	// sample landuse types
	klu, nlu := ksg+len(b.mpr.sg)*nsg, 2
	pn, pincpt, pfimp := make(map[int]float64, len(b.mpr.lu)), make(map[int]float64, len(b.mpr.lu)), make(map[int]float64, len(b.mpr.lu))
	for k, lu := range b.mpr.lu {
		i := -1 /////////////////////////////////////////////
		pn[i] = mann(u[klu+nlu*k])
		pfimp[i] = lu.Fimp
		pincpt[i] = lu.Fimp*dpsto + lu.Ifct*itcp(u[klu+nlu*k+1])
	}

	ksat, n := make(map[int]float64), make(map[int]float64)
	ts := b.frc.h.IntervalSec()
	assignHRUs := func() {
		defer wg.Done()
		var recurs func(int)
		recurs = func(cid int) {
			if _, ok := b.mpr.lu[cid]; !ok {
				log.Fatalf("assignHRUs error, no LandUse assigned to cell ID %d", cid)
			}
			if _, ok := b.mpr.sg[cid]; !ok {
				log.Fatalf("assignHRUs error, no SurfGeo assigned to cell ID %d", cid)
			}
			var hnew hru.HRU
			ksat[cid] = pksat[b.mpr.isg[cid]]
			n[cid] = pn[b.mpr.ilu[cid]]
			drnsto := ppor[b.mpr.isg[cid]] * (1. - pfc[b.mpr.isg[cid]]) * dsoil
			srfsto := ppor[b.mpr.isg[cid]]*pfc[b.mpr.isg[cid]]*dsoil + pincpt[b.mpr.ilu[cid]]
			hnew.Initialize(drnsto, srfsto, pfimp[b.mpr.ilu[cid]], ksat[cid], ts)
			ws[cid] = &hnew
			for _, upcid := range b.strc.t.UpIDs(cid) {
				recurs(upcid)
			}
		}
		recurs(b.cid0)
	}

	wg.Add(1)
	go assignHRUs()

	medQ := b.frc.Q0 * b.strc.a * float64(len(ksat))     // [m/d] to [m³/d]
	gw.New(ksat, b.strc.t, b.strc.w, medQ, 2*medQ, topm) ////////////////////////////// creat new cloner
	wg.Wait()
	return sample{
		ws:   ws,
		gw:   gw,
		p0:   b.buildC0(n, ts), // b.buildFc(f1),
		p1:   b.buildC2(n, ts),
		rill: rill,
	}
}

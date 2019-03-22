package basin

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/maseology/glbopt"
	"github.com/maseology/mmaths"
	"github.com/maseology/mmio"
	"github.com/maseology/objfunc"
	mrg63k3a "github.com/maseology/pnrg/MRG63k3a"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
	"github.com/maseology/goHydro/met"
)

// Basin contais multiple HRUs and forcing data to run independently
type Basin struct {
	frc             *FRC
	mdl             *MDL
	cids            []int
	contarea, fncid float64
	ncid            int
}

type sample struct {
	bsn hru.Basin
	gw  gwru.TMQ
	// tem  tem.TEM
	rill float64
}

func (b *Basin) toSample(rill, m float64) sample {
	h := make(map[int]*hru.HRU, b.ncid)
	for i, v := range b.mdl.b {
		hnew := *v
		hnew.Reset()
		h[i] = &hnew
	}
	return sample{
		bsn:  h,
		gw:   b.mdl.g.Clone(m),
		rill: rill,
	}
}

// Run a single smulation
func Run(ldr *Loader, rill, m float64) float64 {
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
	smpl := b.toSample(rill, m)
	return b.evalWB(&smpl, true)
}

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
	fmt.Println("optimizing..")
	uFinal, _ := glbopt.SCE(10, 2, rng, gen, true)

	p0 := mmaths.LogLinearTransform(0.001, .1, uFinal[0])  // rill storage
	p1 := mmaths.LogLinearTransform(0.001, 10., uFinal[1]) // topmodel m
	fmt.Printf("\nfinal parameters: %v\n", []float64{p0, p1})
	final := b.toSample(p0, p1)
	b.evalWB(&final, true)
}

func (b *Basin) evalWB(p *sample, print bool) (of float64) {
	nstep := b.frc.h.Nstep()
	o, g, x, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
	defer func() {
		of = 1. - objfunc.KGEi(o, s)
		if print {
			// C:/Users/mason/OneDrive/R/dygraph/obssim_csv_viewer.R
			mmio.WriteCSV("hydrograph.csv", "date,obs,sim,gw,excess", dt, o, s, g, x)
			// mmio.ObsSim("hydrograph.png", o[730:], s[730:])
			fmt.Printf("Total number of cells: %d\t %d timesteps\t catchent area: %.3f km²\n", b.ncid, nstep, b.contarea/1000./1000.)
			fmt.Printf("  KGE: %.3f  Bias: %.3f\n", 1.-of, objfunc.Biasi(o, s))
		}
	}()

	// run model
	dtb, dte, intvl := b.frc.h.BeginEndInterval()
	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
		// fmt.Println(d)
		v := b.frc.c[d]
		gwlast := p.gw.Dm
		wbal, asum, rsum, xsum, gsum, ssum, slsum := 0., 0., 0., 0., 0., 0., 0.
		for _, c := range b.cids {
			slast := p.bsn[c].Storage() // initial HRU storage
			slsum += slast
			di := p.gw.GetDi(c)
			if di < -p.rill { // saturation excess runoff (Di: groundwater deficit)
				di += p.rill
				xsum -= di // saturation excess runoff
				gsum += di // negative recharge
			} else {
				di = 0.
			}
			a, r, g := p.bsn[c].Update(v[met.AtmosphericYield]-di, v[met.AtmosphericDemand]*b.mdl.f[c][d.YearDay()-1])
			if a < 0. {
				log.Fatalf(" hru water-balance error, HRU ET = %.3e mm", a*1000.)
			}
			if r < 0. {
				log.Fatalf(" hru water-balance error, HRU runoff = %.3e mm", r*1000.)
			}
			if g < 0. {
				log.Fatalf(" hru water-balance error, HRU potential recharge = %.3e mm", g*1000.)
			}
			s := p.bsn[c].Storage()
			wbal += v[met.AtmosphericYield] - di + slast - (s + g + a + r)
			ssum += s
			asum += a
			rsum += r
			gsum += g
		}
		ssum /= b.fncid
		slsum /= b.fncid
		asum /= b.fncid
		rsum /= b.fncid
		xsum /= b.fncid
		gsum /= b.fncid
		bf := p.gw.Update(gsum) / b.contarea // unit baseflow ([m³/ts] to [m/ts])
		rsum += bf

		if math.Abs(wbal/b.fncid) > 1e-10 {
			fmt.Printf(" pre: %.5f   ex: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, asum, gsum, rsum, v[met.UnitDischarge])
			log.Fatalf(" hru water-balance error, |wbal| = %.3e m", math.Abs(wbal))
		}
		wbalBasin := v[met.AtmosphericYield] - gwlast + slsum - (-p.gw.Dm + ssum + asum + rsum)
		if math.Abs(wbalBasin) > 1e-10 {
			fmt.Printf(" pre: %.5f   ex: %.5f  aet: %.5f  rch: % .5f  sim: %.5f  obs: %.5f\n", v[met.AtmosphericYield], xsum, asum, gsum, rsum, v[met.UnitDischarge])
			fmt.Printf(" stolast: %.5f  sto: %.5f  gwlast: %.5f  gw: %.5f  wbal: % .2e\n", slsum, ssum, gwlast, p.gw.Dm, wbalBasin)
			log.Fatalf(" basin water-balance error, |wbalBasin| = %.3e m", math.Abs(wbalBasin))
		}

		// save results
		dt[i] = d
		o[i] = v[met.UnitDischarge] * b.contarea / 86400.0 // cms
		g[i] = bf * b.contarea / 86400.0
		x[i] = xsum * b.contarea / 86400.0
		s[i] = rsum * b.contarea / 86400.0
		i++
	}
	return
}

func (b *Basin) eval(p *sample) (of float64) {
	nstep := b.frc.h.Nstep()
	o, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
	defer func() {
		of = 1. - objfunc.KGEi(o, s)
	}()

	// run model
	dtb, dte, intvl := b.frc.h.BeginEndInterval()
	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
		v := b.frc.c[d]
		rsum, gsum := 0., 0.
		for _, c := range b.cids {
			di := p.gw.GetDi(c)
			if di < -p.rill { // saturation excess runoff (Di: groundwater deficit)
				di += p.rill
			} else {
				di = 0.
			}
			_, r, g := p.bsn[c].Update(v[met.AtmosphericYield]-di, v[met.AtmosphericDemand]*b.mdl.f[c][d.YearDay()-1])
			rsum += r
			gsum += g
		}
		rsum /= b.fncid
		gsum /= b.fncid
		rsum += p.gw.Update(gsum) / b.contarea // unit baseflow ([m³/ts] to [m/ts])

		// save results
		dt[i] = d
		o[i] = v[met.UnitDischarge] * b.contarea / 86400.0 // cms
		s[i] = rsum * b.contarea / 86400.0
		i++
	}
	return
}

// // Basin contais multiple HRUs and forcing data to run independently
// type Basin struct {
// 	frc                       met.Coll
// 	fhd                       met.Header
// 	bsn                       hru.Basin
// 	gw                        gwru.TMQ
// 	tem                       tem.TEM
// 	sif                       map[int][366]float64
// 	cids                      []int
// 	cellarea, contarea, fncid float64
// 	rill                      float64
// 	ncid                      int
// }

// // NewBasin return a Basin struct
// func NewBasin(ldr *Loader, rill, m float64) Basin {
// 	// import data
// 	frc, mdl := ldr.load(m)
// 	outlet := int(frc.h.Locations[0][0].(int32))
// 	if ldr.outlet > 0 {
// 		outlet = ldr.outlet
// 	}
// 	cids := mdl.t.ContributingAreaIDs(outlet)
// 	ncid := len(cids)
// 	fncid := float64(ncid)
// 	return Basin{
// 		frc:      frc.c,
// 		fhd:      frc.h,
// 		bsn:      mdl.b,
// 		gw:       mdl.g,
// 		tem:      mdl.t.SubSet(outlet),
// 		cids:     cids,
// 		sif:      mdl.f,
// 		cellarea: mdl.a,
// 		ncid:     ncid,
// 		fncid:    fncid,
// 		contarea: mdl.a * fncid, // basin contributing area [m²]
// 		rill:     rill,
// 	}
// }

// // Reset brings the model to an initial state
// func (b *Basin) Reset(rill, m float64) {
// 	b.rill = rill
// 	b.gw.Reset(m)
// 	for _, c := range b.cids {
// 		b.bsn[c].Reset()
// 	}
// }

// // RunCasc basin model with cascading flowpaths
// func (b *Basin) RunCasc() {
// 	nstep := b.fhd.Nstep()
// 	o, s, dt, i := make([]interface{}, nstep), make([]interface{}, nstep), make([]interface{}, nstep), 0
// 	defer func() {
// 		// C:/Users/mason/OneDrive/R/dygraph/obssim_csv_viewer.R
// 		mmio.WriteCSV("hydrograph.csv", "date,obs,sim", dt, o, s)
// 		// mmio.ObsSim("hydrograph.png", o[730:], s[730:])
// 		fmt.Printf("\nTotal number of cells: %d\t %d timesteps\t catchent area: %.3f km²\n", b.ncid, nstep, b.contarea/1000./1000.)
// 	}()

// 	// run model
// 	dtb, dte, intvl := b.fhd.BeginEndInterval()
// 	for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
// 		fmt.Println(d)
// 		v := b.frc[d]
// 		rsum := 0

// 		// save results
// 		dt[i] = d
// 		o[i] = v[met.UnitDischarge]
// 		s[i] = rsum
// 		i++
// 	}
// }

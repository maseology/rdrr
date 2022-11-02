package model

import (
	"math"
	"rdrr/lusg"
	"sort"

	"github.com/maseology/goHydro/hru"
	"github.com/maseology/mmio"
)

const (
	omega    = 1.2
	minFcasc = .0001

	defaultDepSto   = .001  // [m]
	defaultIntSto   = .0005 // [m]
	defaultPorosity = .2    // [-]
	defaultFc       = .3    // [-]
	// defaultSoilDepth = .5    // [m]
)

func (dom *Domain) Parameterize(acasc, soildepth, maxFcasc, dinc, TOPMODELm float64, prnt bool) ([]*Surface, []int, []int, map[int]int, []int) {
	tt := mmio.NewTimer()

	lus := make([]*Surface, dom.Nc)
	xg, xm := make([]int, dom.Nc), make([]int, dom.Nc) // cross-referencing
	// dms := make([]float64, dom.Ngw)                    // mean deficits: state variables for each groundwater zone
	bos := make([]float64, dom.Nc)
	gammas := make([]float64, dom.Ngw) // mean deficits: soil-topologic index (gamma in TOPMODEL)
	m := make(map[int]int, dom.Nc+1)   // cross-referencing cell id to array index
	m[-1] = -1

	mstrm := func() map[int]bool {
		o := make(map[int]bool, len(dom.Mpr.Strms))
		for _, sc := range dom.Mpr.Strms {
			o[sc] = true
		}
		return o
	}() // "has stream" map

	// assign parameters
	for i, c := range dom.Strc.CIDs {
		// cross-referencing
		m[c] = i
		xg[i] = func() int {
			if gid, ok := dom.Mpr.GWx[c]; ok {
				return gid
			}
			panic("gw xr build error")
		}()
		xm[i] = func() int {
			if mid, ok := dom.Frc.XR[c]; ok {
				return mid
			}
			panic("met xr build error")
		}()

		tanbeta := func() float64 {
			if f, ok := dom.Strc.DwnGrad[c]; ok {
				return math.Tan(f)
			}
			panic("fimp load error")
		}()
		ksat := func() float64 {
			if isg, ok := dom.Mpr.SGx[c]; ok {
				if k, ok := dom.Mpr.Ksat[isg]; ok {
					return k
				}
			}
			panic("ksat load error")
		}()
		fimp, ifct := func() (float64, float64) {
			if f, ok := dom.Mpr.Fimp[c]; ok {
				if i, ok := dom.Mpr.Ifct[c]; ok {
					return f, i
				}
			}
			panic("fimp load error")
		}()
		ucasc := func() float64 {
			if s, ok := dom.Strc.DwnGrad[c]; ok {
				if _, ok := mstrm[c]; ok {
					return 1.
				}
				return UcascGaussian(acasc, s)
			}
			panic("fcasc load error")
		}()
		zeta := func() float64 {
			if a, ok := dom.Mpr.Uca[c]; ok {
				zeta := math.Log(a / dom.Strc.Wcell / ksat / tanbeta)
				if math.IsInf(zeta, 0) {
					panic("zeta compute error")
				}
				gammas[xg[i]] += zeta // note: uniform cells
				return zeta
			}
			panic("zeta load error")
		}()
		rzsto, detsto, sma0, det0 := func() (float64, float64, float64, float64) {
			if isw, ok := dom.Mpr.LUx[c]; ok {
				lu := lusg.LandUse{
					DepSto:   defaultDepSto,
					IntSto:   defaultIntSto,
					Porosity: defaultPorosity,
					Fc:       defaultFc,
					Typ:      isw,
				}
				return lu.Rebuild1(soildepth, fimp, ifct)
			}
			panic("detsto load error")
		}()
		bos[i] = func() float64 {
			if _, ok := mstrm[c]; ok {
				return omega * tanbeta * ksat * dom.Frc.IntervalSec
			}
			return 0.
		}()

		h := hru.HRU{Sma: hru.Res{Cap: rzsto, Sto: sma0}, Sdet: hru.Res{Cap: detsto, Sto: det0}, Fimp: fimp, Perc: dom.Frc.IntervalSec * ksat}
		lus[i] = &Surface{
			Hru:   h,
			Fcasc: (maxFcasc-minFcasc)*ucasc + minFcasc,
			Drel:  zeta, // temporarily until gamma is determined
			Dinc:  dinc,
			Bo:    bos[i], //bo, // omega * tanbeta * ksat * dom.Frc.IntervalSec,
			Tm:    TOPMODELm,
		}
	}

	// cross-referencing, for grid outputting
	occx := func() []int {
		o := make([]int, dom.Nc)
		ocids := make([]int, dom.Nc)
		copy(ocids, dom.Strc.CIDs)
		sort.Ints(ocids)
		for i, c := range ocids {
			o[m[c]] = i
		}
		return o
	}() // maps CIDs array index to the ordered cid array index

	// initial state
	for i := 0; i < dom.Ngw; i++ {
		gammas[i] /= dom.Mpr.Fngwc[i]
		// dms[i] = .4
	}
	for i, c := range dom.Strc.CIDs {
		lus[i].Drel = (gammas[dom.Mpr.GWx[c]] - lus[i].Drel) * lus[i].Tm
	}

	if prnt {
		tt.Print("HRUs and TOPMODEL build complete")
	}

	return lus, xg, xm, m, occx // m[dom.Strc.CID0] // -1
}

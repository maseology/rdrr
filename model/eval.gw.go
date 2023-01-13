package model

import (
	"fmt"
	"math"

	"github.com/maseology/mmio"
)

const conv1 = 4. * 365.24 * 1000. // [m/ts] to [mm/yr]

func (dom *Domain) ScreenGroundwater(lus []*Surface, r []float64, fngwc float64, prnt bool) []float64 {
	if prnt {
		tt := mmio.NewTimer()
		defer tt.Print("ScreenGroundwater run complete")
	}
	// fnlus := float64(len(lus))
	// fmt.Println(len(lus), fngwc)

	// test 1: steady to wet conditions
	drainToSteady := func() []float64 {
		dmean := 10.
		rx := r[3] / 30. / 4. / 1000. // [mm/mo] to [m/ts] // april is (generally) the highest, september is the lowest
		for k := 0; k < 1e5; k++ {
			dsum := 0.
			for _, s := range lus {
				d := s.Drel + dmean
				f := math.Exp((s.Dinc - d) / s.Tm)
				if math.IsInf(f, 0) {
					return nil
				}
				hb := s.Bo * f
				dsum += hb
			}

			resid := dsum/fngwc - rx
			if prnt && k%(4*30) == 0 {
				fmt.Printf("  > %6d %6d %10.5f %10.5f %10.1f  (%.1f)\n", k, 4, dmean, dsum/fngwc*conv1, rx*conv1, resid*conv1)
			}
			if math.Abs(resid*conv1) < 0.1 {
				return []float64{resid * conv1, dmean}
			}
			dmean += resid
		}
		return nil
	}()

	if drainToSteady == nil { //|| math.Abs(test1[1]) > 1. {
		return nil
	}
	// return drainToSteady

	// test 2: spring to fall recession
	fallRecession := func(d0 float64) []float64 {
		var resid float64
		im, rr, dmean := 3, 0., d0
		for k := 0; k < 5*30*4; k++ {
			if k%(30*4) == 0 {
				im++
				rr = r[im] / 30. / 4. / 1000. // [mm/mo] to [m/ts]
			}

			dsum := 0.
			for _, s := range lus {
				d := s.Drel + dmean
				f := math.Exp((s.Dinc - d) / s.Tm)
				if math.IsInf(f, 0) {
					return nil
				}
				hb := s.Bo * f
				dsum += hb
			}

			resid = dsum/fngwc - rr
			if prnt && k%4 == 0 {
				fmt.Printf(" >> %6d %6d %10.5f %10.5f %10.1f  (%.1f)\n", k, im+1, dmean, dsum/fngwc*conv1, rr*conv1, resid*conv1)
			}
			dmean += resid
		}
		return []float64{resid * conv1, dmean}
	}(drainToSteady[1])
	return fallRecession
}

// func (dom *Domain) EvaluateGWsaturated(lus []*Surface, rx []float64, cxr map[int]int, xg []int, prnt bool) []float64 {
// 	if prnt {
// 		tt := mmio.NewTimer()
// 		defer tt.Print("EvaluateGWsaturated run complete")
// 	}

// 	// assumption: wettest month is at 0 deficit on average
// 	dss := make([]float64, dom.Ngw)
// 	tms := make(map[float64]int)
// 	hbt := 0.
// 	// for i := range dom.Strc.CIDs {
// 	for _, cid := range dom.Mpr.Strms {
// 		i := cxr[cid]
// 		s := lus[i]
// 		d := s.Drel // dm=0
// 		hb := s.Bo * math.Exp((s.Dinc-d)/s.Tm)
// 		tms[s.Tm]++
// 		dss[xg[i]] += hb
// 		hbt += hb
// 	}
// 	fmt.Println(tms)
// 	fmt.Println(hbt)

// 	resid, sinks, srcs, dsm := 0., 0., 0., 0.
// 	for i, v := range dss {
// 		srcs += rx[i] * dom.Mpr.Fngwc[i] // weighted
// 		sinks += v                       // dom.Mpr.Fngwc[i]
// 		sinkLessSource := v/dom.Mpr.Fngwc[i] - rx[i]
// 		resid += sinkLessSource * dom.Mpr.Fngwc[i] // weighted
// 	}

// 	resid *= 4. * 365.24 * 1000. / float64(dom.Nc)
// 	srcs *= 4. * 365.24 * 1000. / float64(dom.Nc)
// 	sinks *= 4. * 365.24 * 1000. / float64(dom.Nc)
// 	dsm /= float64(dom.Nc)
// 	fmt.Printf("  > %10.5f %10.5f %10.1f  (%.1f)\n", dsm, sinks, srcs, resid)
// 	return []float64{resid} // [mm/year]
// }

// func (dom *Domain) EvaluateGroundwater(lus []*Surface, rx, rn []float64, cxr map[int]int, xg []int, prnt bool) []float64 {
// 	if prnt {
// 		tt := mmio.NewTimer()
// 		defer tt.Print("EvaluateGroundwater run complete")
// 	}

// 	ds := make([]float64, dom.Ngw) // initial water deficits set to saturated (=0) (to be solved for)

// 	// initialize mean d for all locations
// 	resid := func() []float64 {

// 		// douts, fuc := make([]bool, dom.Nc), make([]float64, dom.Nc)
// 		// for i, c := range dom.Strc.CIDs {
// 		// 	fuc[i] = float64(dom.Strc.UpCnt[c])
// 		// 	d := dom.Strc.DwnXR[i]
// 		// 	if d == -1 || xg[i] != xg[d] {
// 		// 		douts[i] = true
// 		// 	}
// 		// }

// 		for k := 0; k < 1e5; k++ {
// 			dss := make([]float64, dom.Ngw)
// 			// for i := range dom.Strc.CIDs {
// 			for _, cid := range dom.Mpr.Strms {
// 				i := cxr[cid]
// 				s := lus[i]
// 				d := s.Drel + ds[xg[i]]
// 				hb := s.Bo * math.Exp((s.Dinc-d)/s.Tm)
// 				dss[xg[i]] += hb
// 			}

// 			resid := 0.
// 			sinks, srcs, dsm := 0., 0., 0.
// 			for i, v := range dss {
// 				srcs += rx[i] * dom.Mpr.Fngwc[i] // weighted
// 				sinks += v                       // dom.Mpr.Fngwc[i]
// 				sinkLessSource := v/dom.Mpr.Fngwc[i] - rx[i]
// 				resid += sinkLessSource * dom.Mpr.Fngwc[i] // weighted
// 				ds[i] += sinkLessSource
// 				dsm += ds[i] * dom.Mpr.Fngwc[i] // weighted
// 			}

// 			if k%(4*15) == 0 {
// 				fmt.Printf("  %8d %10.5f %15.5f %15.5f\n", k, dsm/float64(dom.Nc), sinks*4.*365.24*1000./float64(dom.Nc), srcs*4.*365.24*1000./float64(dom.Nc))
// 			}

// 			resid *= 4. * 365.24 / float64(dom.Nc) // [m/ts]
// 			if math.Abs(resid) <= .001 {
// 				if prnt {
// 					srcs *= 4. * 365.24 * 1000. / float64(dom.Nc)
// 					sinks *= 4. * 365.24 * 1000. / float64(dom.Nc)
// 					dsm /= float64(dom.Nc)
// 					fmt.Printf("  %8d %10.5f %15.5f %15.5f  (%.1f)\n", k, dsm, sinks, srcs, resid*1000.)
// 				}

// 				// ///////////////////////////////////////////
// 				// ///////////////////////////////////////////
// 				// for k := 0; k < 4*365; k++ {
// 				// 	dss := make([]float64, dom.Ngw)
// 				// 	// for i := range dom.Strc.CIDs {
// 				// 	for _, cid := range dom.Mpr.Strms {
// 				// 		i := cxr[cid]
// 				// 		s := lus[i]
// 				// 		d := s.Drel + ds[xg[i]]
// 				// 		hb := s.Bo * math.Exp((s.Dinc-d)/s.Tm)
// 				// 		dss[xg[i]] += hb
// 				// 	}

// 				// 	sinks, srcs, dsm := 0., 0., 0.
// 				// 	// redo := false
// 				// 	for i, v := range dss {
// 				// 		srcs += rx[i] * dom.Mpr.Fngwc[i] // weighted [m/gwzone]
// 				// 		sinks += v                       // [m/gwzone]
// 				// 		// // if !eval[i] && v <= 0. {
// 				// 		// // 	eval[i] = true
// 				// 		// // }
// 				// 		// if !eval[i] && v > 0. && v < rn[i]*float64(dom.Nc) {
// 				// 		// 	eval[i] = true
// 				// 		// 	redo = true
// 				// 		// 	fmt.Printf(" *** %6d (%5.1f) at gwzone ID %d; resid=%.1f\n", k, float64(k)/4./30., i, (v/float64(dom.Nc)-rn[i])*4.*365.24*1000.) // [mm/ts]
// 				// 		// 	// return
// 				// 		// }
// 				// 		ds[i] += v/dom.Mpr.Fngwc[i] - rx[i]
// 				// 		dsm += ds[i] * dom.Mpr.Fngwc[i] // weighted
// 				// 	}

// 				// 	if k%(4*15) == 0 {
// 				// 		srcs *= 4. * 365.24 * 1000. / float64(dom.Nc)
// 				// 		sinks *= 4. * 365.24 * 1000. / float64(dom.Nc)
// 				// 		dsm /= float64(dom.Nc)
// 				// 		fmt.Printf("  %8d %10.5f %15.5f %15.5f\n", k, dsm, sinks, srcs)
// 				// 	}

// 				// 	// if !redo {
// 				// 	// 	fmt.Printf(" *** breaking at %6d ", k)
// 				// 	// 	return
// 				// 	// }
// 				// }
// 				// ///////////////////////////////////////////
// 				// ///////////////////////////////////////////

// 				return []float64{resid}
// 			} //else if resid < 0 {
// 			// 	v := math.Pow10(int(math.Floor(math.Log10(-resid))))
// 			// 	for i := range dss {
// 			// 		ds[i] -= v
// 			// 	}
// 			// } else {
// 			// 	v := math.Pow10(int(math.Floor(math.Log10(resid))))
// 			// 	for i := range dss {
// 			// 		ds[i] += v
// 			// 	}
// 			// }
// 		}
// 		return nil
// 	}()
// 	if resid == nil {
// 		return nil
// 	}

// 	// // eval := make([]bool, dom.Ngw)
// 	// func() { // drain the gw
// 	// 	for k := 0; k < 4*365; k++ {
// 	// 		dss := make([]float64, dom.Ngw)
// 	// 		// for i := range dom.Strc.CIDs {
// 	// 		for _, cid := range dom.Mpr.Strms {
// 	// 			i := cxr[cid]
// 	// 			s := lus[i]
// 	// 			d := s.Drel + ds[xg[i]]
// 	// 			hb := s.Bo * math.Exp((s.Dinc-d)/s.Tm)
// 	// 			dss[xg[i]] += hb
// 	// 		}

// 	// 		sinks, srcs, dsm := 0., 0., 0.
// 	// 		// redo := false
// 	// 		for i, v := range dss {
// 	// 			srcs += rx[i] * dom.Mpr.Fngwc[i] // weighted [m/gwzone]
// 	// 			sinks += v                       // [m/gwzone]
// 	// 			// // if !eval[i] && v <= 0. {
// 	// 			// // 	eval[i] = true
// 	// 			// // }
// 	// 			// if !eval[i] && v > 0. && v < rn[i]*float64(dom.Nc) {
// 	// 			// 	eval[i] = true
// 	// 			// 	redo = true
// 	// 			// 	fmt.Printf(" *** %6d (%5.1f) at gwzone ID %d; resid=%.1f\n", k, float64(k)/4./30., i, (v/float64(dom.Nc)-rn[i])*4.*365.24*1000.) // [mm/ts]
// 	// 			// 	// return
// 	// 			// }
// 	// 			ds[i] += v/dom.Mpr.Fngwc[i] - rx[i]
// 	// 			dsm += ds[i] * dom.Mpr.Fngwc[i] // weighted
// 	// 		}

// 	// 		if k%(4*15) == 0 {
// 	// 			srcs *= 4. * 365.24 * 1000. / float64(dom.Nc)
// 	// 			sinks *= 4. * 365.24 * 1000. / float64(dom.Nc)
// 	// 			dsm /= float64(dom.Nc)
// 	// 			fmt.Printf("  %8d %10.5f %15.5f %15.5f\n", k, dsm, sinks, srcs)
// 	// 		}

// 	// 		// if !redo {
// 	// 		// 	fmt.Printf(" *** breaking at %6d ", k)
// 	// 		// 	return
// 	// 		// }
// 	// 	}
// 	// 	fmt.Println("low flow not reached with a year")
// 	// }()

// 	return resid
// }

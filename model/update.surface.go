package model

import (
	"math"

	"github.com/maseology/goHydro/hru"
)

type Surface struct {
	// ID, GwID               int
	Hru                       hru.HRU
	Fcasc, Drel, Dinc, Bo, Tm float64
}

func (s *Surface) Update(dm, frc, ep float64) (aet, runoff, recharge float64) {

	d := s.Drel + dm

	updateWT := func(p, ep float64, upwardGradient bool) (aet, ro, rch float64) {
		if upwardGradient {
			x := s.Hru.Sma.Sto - s.Hru.Sma.Cap // excess stored (drainable)
			if x < 0. {                        // fill remaining deficit, assume discharge
				rch = x // groundwater discharge (negative recharge)
				x = 0.
			}
			s.Hru.Sma.Sto = s.Hru.Sma.Cap // saturate retention reservoir (drainable porosity)

			// aet, ro, rch = s.Hru.Update(p+x, ep)
			rp := s.Hru.Sdet.Overflow(p + x)                                     // flush detention storage
			sri := s.Hru.Fimp * rp                                               // impervious runoff
			ro = s.Hru.Sma.Overflow(rp-sri) + sri                                // flush retention, compute potential runoff
			avail := s.Hru.Sdet.Overflow(-ep)                                    // remove ep from detention
			avail = s.Hru.Sma.Overflow(avail*(1.-s.Hru.Fimp)) + avail*s.Hru.Fimp // remaining available ep (cannot be >0.)
			aet = ep + avail                                                     // actual et
			// x := h.Sma.Sto - h.Sma.Cap // excess stored (drainable)
			// if x < 0. { // fill remaining deficit, assume discharge
			// 	rch = x
			// 	x = 0.
			// }
			// h.Sma.Sto = h.Sma.Cap // saturate retention reservoir (drainable porosity)
			// ro = h.Sdet.Overflow(p) + x   // fulfill detention reservoir, add excess to runoff
			// avail := h.Sdet.Overflow(-ep) // remove ep from detention
			// // option 1 no evap from gw
			// aet = ep + avail // actual et
			// // // // option 2 unlimited evap from gw
			// // // rch += avail // ep assumed unlimited from a saturated surface (Note: avail cannot be >0.)
			// // // aet = ep     // completely satisfied over a high watertable
			// // // option 3 limited evap
			// // dh := h.Perc * math.Exp(dwt) * (1. - h.Fimp)
			// // if -avail > dh { // (Note: avail and dh cannot be >0.)
			// // 	avail += dh      // remaining available ep (cannot be >0.)
			// // 	rch -= dh        // (Note: dh cannot be >0.)
			// // 	aet = ep + avail // actual et
			// // } else {
			// // 	rch += avail // ep assumed unlimited from a saturated surface (Note: avail cannot be >0.)
			// // 	aet = ep     // completely satisfied over a high watertable
			// // }
			// // // option 4 limited evap 2
			// // dwt *= (1. - h.Fimp)
			// // if avail > dwt {
			// // 	rch += avail // ep assumed unlimited from a saturated surface (Note: avail cannot be >0.)
			// // 	aet = ep     // completely satisfied over a high watertable
			// // } else {
			// // 	avail -= dwt     // remaining available ep (cannot be >0.)
			// // 	rch += dwt       // (Note: dwt cannot be >0.)
			// // 	aet = ep + avail // actual et
			// // }
		} else {
			aet, ro, rch = s.Hru.Update(p, ep)
		}
		return
	}

	a, r, g := updateWT(frc, ep, d < 0.) // false) // integration disabled /////////////////////////////////////////////////////

	s.Hru.Sdet.Sto += r * (1. - s.Fcasc)
	r *= s.Fcasc
	// g += s.Hru.InfiltrateSurplus()

	hb := 0.
	if s.Bo > 0. {
		hb = s.Bo * math.Exp((s.Dinc-d)/s.Tm)
		r += hb
	}

	// fmt.Printf("  a = %.4f;  r = %.4f;  b = %.4f;  g = %.4f;  s = %.4f\n", a, r-hb, hb, g, s.Hru.Storage())

	return a, r, g - hb
	// fmt.Printf("do something at surface %d with frc = %.3f, ep = %.3f and dm = %.3f\n", s.ID, frc, ep, *dm)
}

// func (s *Surface) SpinUp(done <-chan interface{}, dm *float64, chanSource <-chan float64, nSources int) chan float64 {
// 	out := make(chan float64)
// 	go func() {
// 		cnt, frc, ep := 0, 0., 0.
// 		for {
// 			select {
// 			case <-done:
// 				fmt.Printf("surface %d done\n", s.ID)
// 				close(out)
// 				return
// 			case v := <-chanSource: // adding sources/sinks
// 				if v < 0. {
// 					ep -= v
// 				} else {
// 					frc += v
// 				}
// 				cnt++
// 				if cnt == nSources {
// 					// // fmt.Printf("do something at surface %d with frc = %.3f and dm = %.3f  ::", s.ID, frc, *dm)

// 					// _, r, g := s.Hru.UpdateWT(frc, ep, false) // *dm+s.Drel < 0.)

// 					// s.Hru.Sdet.Sto += r * (1. - s.Fcasc)
// 					// r *= s.Fcasc
// 					// g += s.Hru.InfiltrateSurplus()

// 					// hb := 0.
// 					// if s.Qstrm > 0. {
// 					// 	hb = s.Qstrm * math.Exp(-(*dm+s.Drel)/s.Tm)
// 					// 	r += hb
// 					// }
// 					// // fmt.Printf("  a = %.4f;  r = %.4f;  b = %.4f;  g = %.4f;  s = %.4f\n", a, r-hb, hb, g, s.Hru.Storage())

// 					// frc = 0.
// 					// ep = 0.
// 					// cnt = 0
// 					// out <- r
// 					// *dm += (hb - g) / s.Fngw
// 					// fmt.Printf("do something at surface %d with frc = %.3f, ep = %.3f and dm = %.3f\n", s.ID, frc, ep, *dm)
// 					cnt = 0
// 					out <- frc - ep
// 					frc, ep = 0., 0.
// 				}
// 			}
// 		}
// 	}()
// 	return out
// }

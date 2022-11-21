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

func (s *Surface) Update(dm, kin, ya, ep float64) (aet, runoff, recharge float64) {

	d := s.Drel + dm

	updateWT := func(p, ep, def float64) (aet, ro, rch float64) {
		if def <= 0 { // TOPMODEL is super-saturated/high potential for discharge into SMA

			// seep to sma (upwelling)
			negrch := s.Hru.Perc * def // groundwater discharge (negative recharge)
			s.Hru.Sma.Sto -= negrch
			aet, ro, rch = s.Hru.Update(p, ep)
			rch += negrch

			// // satisfy evaporation demand
			// _, ro, rch = s.Hru.Update(p, 0.) // (ep is used up)
			// rch += negrch - ep               // gw exchange + groundwater evaporation=negative recharge
			// aet = ep

			// x := s.Hru.Sma.Sto - s.Hru.Sma.Cap // excess stored (drainable)
			// negrch := 0.

			// // // saturate retention reservoir (drainable porosity)
			// // if x < 0. { // fill remaining deficit, assume discharge
			// // 	negrch = x // groundwater discharge (negative recharge)
			// // 	x = 0.
			// // }
			// // s.Hru.Sma.Sto = s.Hru.Sma.Cap

			// // satisfy evaporation demand
			// if x < 0. { // sma deficit
			// 	if ep >= -x {
			// 		negrch = x // groundwater evaporation (negative recharge)
			// 		ep += x
			// 		x = 0.
			// 		s.Hru.Sma.Sto = s.Hru.Sma.Cap
			// 	} else {
			// 		negrch = ep // groundwater evaporation (negative recharge)
			// 		x += ep
			// 		ep = 0.

			// 	}
			// }

			// // // upwelling
			// // if x < 0. { // fill remaining deficit, assume discharge
			// // 	dh := -x * (1. - math.Exp(-s.Hru.Perc))
			// // 	negrch = -dh
			// // 	x += dh
			// // }

			// // rp := s.Hru.Sdet.Overflow(p + x)                                     // flush detention storage
			// // sri := s.Hru.Fimp * rp                                               // impervious runoff
			// // ro = s.Hru.Sma.Overflow(rp-sri) + sri                                // flush retention, compute potential runoff
			// // avail := s.Hru.Sdet.Overflow(-ep)                                    // remove ep from detention
			// // avail = s.Hru.Sma.Overflow(avail*(1.-s.Hru.Fimp)) + avail*s.Hru.Fimp // remaining available ep (cannot be >0.)
			// // aet = ep + avail                                                     // actual et
			// aet, ro, rch = s.Hru.Update(p+x, ep)
			// rch += negrch // adding negative recharge
		} else {
			aet, ro, rch = s.Hru.Update(p, ep)
		}
		return
	}

	if s.Bo > 0. { // stream cells
		a, r, g := updateWT(ya, ep, d)
		hb := s.Bo * math.Exp((s.Dinc-d)/s.Tm)
		r += kin + hb
		return a, r, g - hb
	} else {
		a, r, g := updateWT(kin+ya, ep, d) // false)
		s.Hru.Sdet.Sto += r * (1. - s.Fcasc)
		r *= s.Fcasc
		g += s.Hru.InfiltrateSurplus() // stops cascade towers
		return a, r, g
	}

	//

	// hb := 0.
	// if s.Bo > 0. {
	// 	hb = s.Bo * math.Exp((s.Dinc-d)/s.Tm)
	// 	r += hb
	// }

	// fmt.Printf("  a = %.4f;  r = %.4f;  b = %.4f;  g = %.4f;  s = %.4f\n", a, r-hb, hb, g, s.Hru.Storage())

	// fmt.Printf("do something at surface %d with frc = %.3f, ep = %.3f and dm = %.3f\n", s.ID, frc, ep, *dm)
	// return a, r, g - hb
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

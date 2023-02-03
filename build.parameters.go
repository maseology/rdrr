package rdrr

import (
	"log"
	"math"
)

const (
	soildepth = .3
	porosity  = .3
	fc        = .1
	depsto    = .001
	intsto    = .001
)

func buildParameters(s *Structure, mp *Mapper) Parameter {
	zetas, ucas, tanbetas, ksats, depstos := make([]float64, s.Nc), make([]float64, s.Nc), make([]float64, s.Nc), make([]float64, s.Nc), make([]float64, s.Nc)
	gammas := make([]float64, len(mp.Fngwc))
	for i := range s.Cids {
		ksats[i] = mp.Ksat[i]
		zetas[i], tanbetas[i], ucas[i] = func() (float64, float64, float64) {
			tanbeta := math.Tan(s.Dwngrad[i])
			uca := float64(s.Upcnt[i]) * s.GD.Cwidth
			zeta := math.Log(uca / ksats[i] / tanbeta)
			if math.IsInf(zeta, 0) {
				panic("zeta compute error")
			}
			gammas[mp.Igw[i]] += zeta // note: uniform cells
			return zeta, tanbeta, uca
		}()
		depstos[i] = func() float64 {
			s := soildepth*porosity*fc + mp.Fimp[i]*depsto + intsto*mp.Ifct[i]
			switch mp.Ilu[i] {
			case Channel:
				// rzsto = 0.
				s = 0.
			case Waterbody, Lake: // Open water
				s = soildepth
			case Noflow:
				s = math.MaxFloat64
			case Urban: // (assumed drained/serviced)
				s = soildepth*porosity*fc*(1.-mp.Fimp[i]) + mp.Fimp[i]*depsto + intsto*mp.Ifct[i]
			case ShortVegetation, TallVegetation, Forest, Swamp, Wetland, SparseVegetation, DenseVegetation, Agriculture, Meadow, Marsh, Barren:
				// do nothing
			default:
				log.Fatalf(" buildParameters.depstos: no value assigned to ID %d", mp.Ilu[i])
			}
			return s
		}()
	}
	for i := 0; i < len(mp.Fngwc); i++ {
		if mp.Fngwc[i] > 0. {
			gammas[i] /= mp.Fngwc[i]
		}
	}
	return Parameter{zetas, ucas, tanbetas, ksats, depstos, gammas}
}

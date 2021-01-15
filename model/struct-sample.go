package model

import (
	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
)

// sample is a parameterized subdomain
type sample struct {
	ws    hru.WtrShd        // hru watershed
	gw    map[int]*gwru.TMQ // topmodel
	cascf map[int]float64   // cascade fraction
	dir   string
	// swsr, celr, p0, p1 map[int]float64
}

// func (s *sample) copy() sample {
// 	return sample{
// 		ws:    hru.CopyWtrShd(s.ws),
// 		cascf: mmio.CopyMapif(s.cascf),
// 		gw: func(origTMQ map[int]*gwru.TMQ) map[int]*gwru.TMQ {
// 			newTMQ := make(map[int]*gwru.TMQ, len(origTMQ))
// 			for k, v := range origTMQ {
// 				cpy := v.Copy()
// 				newTMQ[k] = &cpy
// 			}
// 			return newTMQ
// 		}(s.gw),
// 	}
// }

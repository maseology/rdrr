package basin

import (
	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
)

type sample struct {
	ws     hru.WtrShd // hru watershed
	gw     gwru.TMQ   // topmodel
	p0, p1 map[int]float64
	rill   float64
}

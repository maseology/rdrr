package prep

import (
	"github.com/maseology/goHydro/snowpack"
	"github.com/maseology/goHydro/solirrad"
)

// Cell is a computational element needed to compute yields
type Cell struct {
	SI solirrad.SolIrad
	SP snowpack.CCF
}

// NewCell creates a new cell struct
func NewCell(LatitudeDeg, SlopeRad, AspectCwnRad float64) Cell {
	return Cell{
		SI: solirrad.New(LatitudeDeg, SlopeRad, AspectCwnRad),
		SP: snowpack.NewDefaultCCF(),
	}
}

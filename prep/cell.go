package prep

import (
	"github.com/maseology/goHydro/snowpack"
	"github.com/maseology/goHydro/solirrad"
)

// Cell is a computational element needed to compute yields
type Cell struct {
	SI                solirrad.SolIrad
	SP                snowpack.CCF
	b, g, alpha, beta float64
}

// NewCell creates a new cell struct
func NewCell(LatitudeDeg, SlopeRad, AspectCwnRad, b, g, alpha, beta, tindex, ddfc, baseT, tsf float64) Cell {
	return Cell{
		SI:    solirrad.New(LatitudeDeg, SlopeRad, AspectCwnRad),
		SP:    snowpack.NewCCF(tindex, 0.0045, ddfc, baseT, tsf),
		b:     b,
		g:     g,
		alpha: alpha,
		beta:  beta,
	}
}

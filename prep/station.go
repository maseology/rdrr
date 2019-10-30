package prep

import "github.com/maseology/goHydro/solirrad"

// Location is a point in space where computations are made
type Location struct {
	SI solirrad.SolIrad
}

func newLocation(LatitudeDeg, SlopeRad, AspectCwnRad float64) Location {
	return Location{
		SI: solirrad.New(LatitudeDeg, SlopeRad, AspectCwnRad),
	}
}

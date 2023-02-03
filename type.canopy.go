package rdrr

const ( // canopy types
	open = iota
	shrub
	coniferous
	deciduous
	mixedVegetation
)

// relativeCover creates a canopy cover factor based on land use
func relativeCover(canopyID, surfaceID int) float64 {
	f := 0.
	switch canopyID {
	case coniferous, deciduous, mixedVegetation:
		f += 1.
	case shrub:
		f += .5
	}
	switch surfaceID {
	case DenseVegetation:
		f += 1.25
	case ShortVegetation, TallVegetation, Forest, Swamp:
		f += 1.
	case Agriculture, Meadow:
		f += .85
	case Wetland, Marsh, SparseVegetation:
		f += .35
	}
	return f
}

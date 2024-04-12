package rdrr

// surface types
const (
	Noflow = iota
	Waterbody
	ShortVegetation
	TallVegetation
	Urban
	Agriculture // 5
	Forest
	Meadow
	Wetland
	Swamp
	Marsh // 10
	Channel
	Lake
	Barren
	SparseVegetation
	DenseVegetation
)

type SurfaceSet struct {
	Ilu, Ulu, Icov []int
	Fimp, Fint     []float64
}

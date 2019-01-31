package gwru

// GWmodel interface to groundwater model
type GWmodel interface {
	New(p ...float64)
	Update(g float64) float64
}

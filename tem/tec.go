package tem

// TEC topologic elevation model cell
type TEC struct {
	Z, S, A float64
	ds      int
}

// New constructor
func (t *TEC) New(z, s, a float64, ds int) {
	t.Z = z   // elevation
	t.S = s   // slope
	t.A = a   // aspect
	t.ds = ds // downslope id
}

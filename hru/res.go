package hru

// res simple linear reservoir
type res struct {
	sto float64
	cap float64
}

func (r *res) overflow(p float64) float64 {
	r.sto += p
	if r.sto < 0 {
		d := r.sto
		r.sto = 0.0
		return d
	} else if r.sto > r.cap {
		d := r.cap - r.sto
		r.sto = r.cap
		return d
	} else {
		return 0.0
	}
}

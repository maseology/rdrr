package hru

// bit-wise status flag
const (
	snowOnGround = 1 << iota
	waterOnSurface
	availPoreWater
	deficitPoreWater
)

// params : parameter set
type params struct {
	fimp float32
	fc   float32
	perc float32
}

// res : simple linear reservoir
type res struct {
	sto float32
	cap float32
}

func (r *res) overflow(p float32) float32 {
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

// HRU : the Hydrologic Response Unit
type HRU struct {
	par  params
	sma  res
	stat byte
}

// Initialize : HRU constructor
func (h *HRU) Initialize(cap, fimp, fc, ksat, ts float32) {
	h.sma.sto = 0.0
	h.sma.cap = cap
	h.par.fimp = fimp
	h.par.fc = fc
	h.par.perc = ts * (1.0 - fimp) * ksat // gravity-driven percolation rate m/6hr
}

// Update : hru given a set of forcings
func (h *HRU) Update(p, pet float32) (aet, ro, rch float32) {
	ro = h.sma.overflow(p)
	aet = h.sma.overflow(-pet)
	rch = h.sma.overflow(-h.par.perc)
	h.updateStatus()
	return
}

func (h *HRU) updateStatus() {
	if h.sma.sto < h.sma.cap {
		h.stat |= deficitPoreWater
		if h.sma.sto > 0 {
			h.stat |= availPoreWater
		} else {
			h.stat &^= availPoreWater
		}
	} else {
		h.stat &^= deficitPoreWater
		if h.sma.sto > h.sma.cap {
			h.stat |= waterOnSurface
		} else {
			h.stat &^= waterOnSurface
		}
	}
}

package hru

// bit-wise status flag
const (
	snowOnGround = 1 << iota
	waterOnSurface
	availPoreWater
	deficitPoreWater
)

// Basin set of HRUs
type Basin map[int]HRU

// HRU : the Hydrologic Response Unit
type HRU struct {
	par  params
	sma  res
	stat byte
}

// Initialize : HRU constructor
func (h *HRU) Initialize(cap, fimp, fc, ksat, ts float64) {
	h.sma.sto = 0.0
	h.sma.cap = cap
	h.par.fimp = fimp
	h.par.fc = fc
	h.par.perc = ts * (1.0 - fimp) * ksat // gravity-driven percolation rate m/ts
}

// Update : hru given a set of forcings
func (h *HRU) Update(p, pet float64) (aet, ro, rch float64) {
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

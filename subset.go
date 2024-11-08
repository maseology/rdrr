package rdrr

import "sort"

func subsetByGWzones(s *Structure, w *Subwatershed, m *Mapper, p *Parameter, gwids []int) {

	isIn := func(gi int) bool {
		for _, g := range gwids {
			if gi == g {
				return true
			}
		}
		return false
	}
	newCids, newA, mA, mst := []int{}, []int{}, make(map[int]int), make(map[int]int)
	for a, c := range s.Cids {
		if isIn(m.Igw[a]) {
			mA[a] = len(newCids)
			newCids = append(newCids, c)
			newA = append(newA, a)
			mst[w.Sid[a]]++
		}
	}
	s.GD.ResetActives(newCids)
	nc := len(newA)

	newFngwc, newGamma, mg := make([]float64, len(gwids)), make([]float64, len(gwids)), make(map[int]int)
	for i, g := range gwids {
		mg[g] = i
		newFngwc[i] = m.Fngwc[g]
		newGamma[i] = p.Gamma[g]
	}
	ms := func() map[int]int {
		l := make([]int, 0, len(mst))
		for i := range mst {
			l = append(l, i)
		}
		sort.Ints(l)
		m := make(map[int]int, len(l))
		for i, s := range l {
			m[s] = i
		}
		return m
	}()
	newDnslp, newDs := make([]float64, nc), make([]int, nc)
	newIlu, newIsg, newIgw, newIcov := make([]int, nc), make([]int, nc), make([]int, nc), make([]int, nc)
	newKsat, newFimp, newFint := make([]float64, nc), make([]float64, nc), make([]float64, nc)
	newZeta, newUca, newTanbeta, newDepSto, newDrel := make([]float64, nc), make([]float64, nc), make([]float64, nc), make([]float64, nc), make([]float64, nc)
	newSid := make([]int, nc)
	for i, a := range newA {
		newDnslp[i] = s.Dwnslope[a]
		newDs[i] = mA[s.Ds[a]]
		newIlu[i] = m.Ilu[a]
		newIsg[i] = m.Isg[a]
		newIgw[i] = mg[m.Igw[a]] // remapped to 0-based array
		newIcov[i] = m.Icov[a]
		newKsat[i] = m.Ksat[a]
		newFimp[i] = m.Fimp[a]
		newFint[i] = m.Fint[a]
		newZeta[i] = p.Zeta[a]
		newUca[i] = p.Uca[a]
		newTanbeta[i] = p.Tanbeta[a]
		newDepSto[i] = p.DepSto[a]
		newDrel[i] = p.Drel[a]
		newSid[i] = ms[w.Sid[a]]
	}

	func() {
		upcnt := func() []int {
			u := make(map[int]int, nc)
			for i := range newA {
				u[i] = 1 // initialize
			}
			for i := range newA { // newA already ordered topologically
				if newDs[i] > -1 {
					u[newDs[i]] += u[i]
				}

			}
			uc := make([]int, nc)
			for i, c := range u {
				uc[i] = c
			}
			return uc
		}

		s.Cids = newCids      // topologically safe ordered grid cell ids
		s.Dwnslope = newDnslp // steepest cell slope
		s.Ds = newDs          // down slope cell index
		s.Upcnt = upcnt()     // count of upslope cells
		s.Nc = nc             // number of model cells
	}()

	func() {
		newScis, newSds := make([][]int, len(ms)), make([][]int, len(ms))
		newIsws, newSgw := make([]int, len(ms)), make([]int, len(ms))
		newFnsc := make([]float64, len(ms))
		newIslake := make([]bool, len(ms))
		newDsws := make([]SWStopo, len(ms))
		for s, i := range ms {
			tScis := make([]int, len(w.Scis[s]))
			tSds := make([]int, len(w.Sds[s]))
			for i, c := range w.Scis[s] {
				tScis[i] = mA[c]
				tSds[i] = w.Sds[s][i]
			}
			newScis[i] = tScis
			newSds[i] = tSds
			newIsws[i] = w.Isws[s]
			newSgw[i] = mg[w.Sgw[s]] // remapped to 0-based array
			newFnsc[i] = w.Fnsc[s]
			newIslake[i] = w.Islake[s]
			newDsws[i] = func() SWStopo {
				if w.Dsws[s].Sid <= -1 {
					return SWStopo{-1, -1}
				} else if _, ok := ms[w.Dsws[s].Sid]; !ok {
					return SWStopo{-1, -1}
				}
				return SWStopo{Sid: ms[w.Dsws[s].Sid], Cid: w.Dsws[s].Cid}
			}()
		}

		w.Outer = nil
		w.Scis = newScis // set of cell indices per sws
		w.Sds = newSds   // cell topology per sub-watershed, <0 is routed to down-SWS
		w.Dsws = newDsws // [downslope sub-watershed,cell index receiving input], -1 out of model
		w.Sid = newSid   // 0-based cell index to 0-based sws index
		w.Isws = newIsws // sws index to original sub-watersed ID (needed for forcings)
		w.Sgw = newSgw   // sws index to GW zone
		w.Fnsc = newFnsc // number of cells per sws
		w.Islake = newIslake
		w.Ns = len(ms)
		w.buildComputationalOrder1()
	}()

	func() {
		newmx := make(map[int]int, len(newCids))
		for i, c := range newCids {
			newmx[c] = i
		}
		m.Mx = newmx
		m.Ilu = newIlu
		m.Isg = newIsg
		m.Igw = newIgw
		m.Icov = newIcov
		m.Ksat = newKsat
		m.Fimp = newFimp
		m.Fint = newFint
		m.Fngwc = newFngwc
	}()

	func() {
		p.Zeta = newZeta
		p.Uca = newUca
		p.Tanbeta = newTanbeta
		p.DepSto = newDepSto
		p.Drel = newDrel
		p.Gamma = newGamma
	}()
}

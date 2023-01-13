package model

// FixStreamUca resets UCA in streamcells to the largest upslope UCA that is not a stream cell
// prevents very negative UCAs occuring in large watersheds covering many gw zones.
func (dom *Domain) FixStreamUca() {
	newUca := make(map[int]float64, len(dom.Mpr.Uca))
	for c, u := range dom.Mpr.Uca {
		newUca[c] = u
	}
	isstrm := make(map[int]bool, len(dom.Mpr.Strms))
	queue := make([]int, len(dom.Mpr.Strms))
	for i, sc := range dom.Mpr.Strms {
		queue[i] = sc
		isstrm[sc] = false
	}

	for len(queue) > 0 {
		sc := queue[0]
		queue = queue[1:]

		ax := 0.
		for _, uc := range dom.Strc.UpSlps[sc] {
			if b, ok := isstrm[uc]; b || !ok {
				if a, ok := dom.Mpr.Uca[uc]; ok {
					if a > ax {
						ax = a
					}
				} else {
					panic("zeta load error 0001")
				}
			}
		}
		if ax > 0. {
			newUca[sc] = ax
			isstrm[sc] = true
		} else {
			queue = append(queue, sc)
		}
	}

	dom.Mpr.Uca = newUca
}

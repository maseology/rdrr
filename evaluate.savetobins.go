package rdrr

func (ev *Evaluator) saveToBins(rel []*realization, sdm, monq, hyd []float64, nt int, outdirprfx string) {

	if ev.Mons != nil && len(monq) > 0 {
		// writeFloats(outdirprfx+"mon.bin", monq)
		writeMons(outdirprfx+"mon.gob", ev.Mons, monq, nt)
	}

	ev.saveToRasterBins(rel, outdirprfx)
	writeFloats(outdirprfx+"sdm.bin", sdm)
	writeFloats(outdirprfx+"hyd.bin", hyd)
}

func (ev *Evaluator) saveToRasterBins(rel []*realization, outdirprfx string) {
	if rel[0].coll != nil {
		nc := ev.Nc
		spr, sae, sro, srch, lsto := make([]float64, nc*12), make([]float64, nc*12), make([]float64, nc*12), make([]float64, nc*12), make([]float64, nc)
		for k, aids := range ev.Saids {
			relk := rel[k]
			nsc := len(aids)
			for m := range 12 {
				for i, a := range aids {
					spr[m*nc+a] = relk.coll.spr[m*nsc+i]
					sae[m*nc+a] = relk.coll.sae[m*nsc+i]
					sro[m*nc+a] = relk.coll.sro[m*nsc+i]
					srch[m*nc+a] = relk.coll.srch[m*nsc+i]
					if m == 0 {
						lsto[a] = relk.x[i].Sto
					}
				}
			}
		}

		writeFloats(outdirprfx+"spr.bin", spr)
		writeFloats(outdirprfx+"sae.bin", sae)
		writeFloats(outdirprfx+"sro.bin", sro)
		writeFloats(outdirprfx+"srch.bin", srch)
		writeFloats(outdirprfx+"lsto.bin", lsto)
	}
}

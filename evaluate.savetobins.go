package rdrr

func (ev *Evaluator) saveToBins(rel []*realization, sdm, monq, hyd []float64, nt int, outdirprfx string) {

	if ev.Mons != nil {
		// writeFloats(outdirprfx+"mon.bin", monq)
		writeMons(outdirprfx+"mon.gob", ev.Mons, monq, nt)
	}

	nc := ev.Nc
	spr, sae, sro, srch, lsto := make([]float64, nc*12), make([]float64, nc*12), make([]float64, nc*12), make([]float64, nc*12), make([]float64, nc)
	for k, cids := range ev.Scids {
		relk := rel[k]
		nsc := len(cids)
		for m := range 12 {
			for i, c := range cids {
				spr[m*nc+c] = relk.spr[m*nsc+i]
				sae[m*nc+c] = relk.sae[m*nsc+i]
				sro[m*nc+c] = relk.sro[m*nsc+i]
				srch[m*nc+c] = relk.srch[m*nsc+i]
				if m == 0 {
					lsto[c] = relk.x[i].Sto
				}
			}
		}
	}

	writeFloats(outdirprfx+"spr.bin", spr)
	writeFloats(outdirprfx+"sae.bin", sae)
	writeFloats(outdirprfx+"sro.bin", sro)
	writeFloats(outdirprfx+"srch.bin", srch)
	writeFloats(outdirprfx+"lsto.bin", lsto)
	writeFloats(outdirprfx+"sdm.bin", sdm)
	writeFloats(outdirprfx+"hyd.bin", hyd)
}

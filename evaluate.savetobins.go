package rdrr

func (ev *Evaluator) saveToBins(rel []*basin, monq, sdm, stage, hyd []float64, outdirprfx string) {

	if ev.Nm > 0 && len(monq) > 0 {
		// writeFloats(outdirprfx+"mon.bin", monq)
		writeMons(outdirprfx+"mon.gob", monq)
	}

	writeFloats(outdirprfx+"sdm.bin", sdm)
	writeFloats(outdirprfx+"hyd.bin", hyd)
	writeFloats(outdirprfx+"stage.bin", stage)
	if rel[0].coll != nil {
		ev.saveToRasterBins(rel, outdirprfx)
	}
}

func (ev *Evaluator) saveToRasterBins(rel []*basin, outdirprfx string) {
	nc := ev.Nc
	spr, sae, sro, srch, sdch, lsto := make([]float64, nc*12), make([]float64, nc*12), make([]float64, nc*12), make([]float64, nc*12), make([]float64, nc*12), make([]float64, nc)
	for k, aids := range ev.Sais {
		relk := rel[k]
		nsc := len(aids)
		for m := range 12 {
			for i, a := range aids {
				spr[m*nc+a] = relk.coll.spr[m*nsc+i]
				sae[m*nc+a] = relk.coll.sae[m*nsc+i]
				sro[m*nc+a] = relk.coll.sro[m*nsc+i]
				srch[m*nc+a] = relk.coll.srch[m*nsc+i]
				sdch[m*nc+a] = relk.coll.sdch[m*nsc+i]
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
	writeFloats(outdirprfx+"sdch.bin", sdch)
	writeFloats(outdirprfx+"lsto.bin", lsto)
}

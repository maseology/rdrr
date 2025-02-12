package rdrr

import "math"

func (ev *Evaluator) EvaluateSteady(strc *Structure, steadyPre float64, outdirprfx string) {

	// prep
	ng := len(ev.Fngwc)
	rel, _, _, _, _ := ev.buildRealization(1, ng, true)

	dms, dmsv := make([]float64, ng), make([]float64, ng)
	qlast := math.MaxFloat64
	k := 0
	for {
		for ig := 0; ig < ng; ig++ {
			dms[ig] += dmsv[ig]
			dmsv[ig] = 0.
		}

		relk, gi := rel[k], ev.Sgw[k]
		_, q, dd := relk.rdrr(steadyPre, 0., dms[gi]/ev.M[gi], 0, 0, k)
		dmsv[gi] += dd
		if math.Abs(qlast-q) < 0.001 {
			break
		}
		qlast = q
	}

	spr, sae, sro, srch, lsto := make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc), make([]float64, ev.Nc)
	relk := rel[k]
	for i, a := range ev.Saids[k] {
		spr[a] = relk.coll.spr[i]
		sae[a] = relk.coll.sae[i]
		sro[a] = relk.coll.sro[i]
		srch[a] = relk.coll.srch[i]
		lsto[a] = relk.x[i].Sto
	}

	gspr := strc.GD.NullFloat32(-9999.)
	gsae := strc.GD.NullFloat32(-9999.)
	gsro := strc.GD.NullFloat32(-9999.)
	gsrch := strc.GD.NullFloat32(-9999.)
	glsto := strc.GD.NullFloat32(-9999.)
	for i, c := range strc.Cids {
		gspr[c] = float32(spr[i])
		gsae[c] = float32(sae[i])
		gsro[c] = float32(sro[i])
		gsrch[c] = float32(srch[i])
		glsto[c] = float32(lsto[i])
	}

	strc.GD.ToBIL(outdirprfx+"spr.bil", gspr)
	strc.GD.ToBIL(outdirprfx+"sae.bil", gsae)
	strc.GD.ToBIL(outdirprfx+"sro.bil", gsro)
	strc.GD.ToBIL(outdirprfx+"srch.bil", gsrch)
	strc.GD.ToBIL(outdirprfx+"lsto.bil", glsto)
}

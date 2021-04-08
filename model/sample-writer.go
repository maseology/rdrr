package model

import "github.com/maseology/mmio"

func (s *sample) write(dir string) error {
	mmio.WriteRMAP(dir+"s.cascf.rmap", s.cascf, false)
	mmio.DeleteFile(dir + "s.gw.drel.rmap")
	mmio.DeleteFile(dir + "s.gw.Qs.rmap")
	// mmio.DeleteFile(dir + "s.gw.g-ti.rmap")
	for _, v := range s.gw {
		mmio.WriteRMAP(dir+"s.gw.drel.rmap", v.D, true)
		mmio.WriteRMAP(dir+"s.gw.Qs.rmap", v.Qs, true)
		// mmio.WriteRMAP(dir+"s.gw.g-ti.rmap", v.RelTi(), true) // = drel/m
	}
	perc, fimp, smacap, srfcap := make(map[int]float64, len(s.ws)), make(map[int]float64, len(s.ws)), make(map[int]float64, len(s.ws)), make(map[int]float64, len(s.ws))
	for c, h := range s.ws {
		perc[c], fimp[c], smacap[c], srfcap[c] = h.Perc, h.Fimp, h.Sma.Cap, h.Sdet.Cap
	}
	mmio.WriteRMAP(dir+"s.ws.perc.rmap", perc, false)
	mmio.WriteRMAP(dir+"s.ws.fimp.rmap", fimp, false)
	mmio.WriteRMAP(dir+"s.ws.smacap.rmap", smacap, false)
	mmio.WriteRMAP(dir+"s.ws.sdetcap.rmap", srfcap, false)
	return nil
}

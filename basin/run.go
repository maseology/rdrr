package basin

import (
	"fmt"
	"log"
)

// Run a single simulation with water balance checking
func Run(ldr *Loader, u []float64) float64 {
	// d := newDomain(ldr)
	d := newUniformDomain(ldr)
	b := d.newSubDomain(ldr.Outlet)
	fmt.Printf(" building sample HRUs and TOPMODEL\n\n")
	// smpl := b.toDefaultSample(topm, mann)
	smpl := b.toSampleU(u...)

	for _, c := range b.cids {
		if smpl.ws[c] == nil {
			log.Fatalln(" basin.Run() error: nil hru")
		}
	}
	b.printParam(u...)
	return b.evalCascWB(&smpl, 0., true)
}

// RunDefault runs simulation with default parameters
func RunDefault(ldr *Loader, topQo, topm, fcasc, freeboard float64) float64 {
	d := newDomain(ldr)
	b := d.newSubDomain(ldr.Outlet)
	fmt.Printf(" catchment area: %.1f km²\n", b.contarea/1000./1000.)
	fmt.Printf(" building sample HRUs and TOPMODEL\n\n")
	smpl := b.toDefaultSample(topQo, topm, fcasc)

	for _, c := range b.cids {
		if smpl.ws[c] == nil {
			log.Fatalln(" basin.Run() error: nil hru")
		}
	}

	fmt.Printf(" running model..\n\n")
	return b.evalCascWB(&smpl, freeboard, true)
}

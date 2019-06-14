package basin

import (
	"fmt"
	"log"
)

// Run a single simulation with water balance checking
func Run(ldr *Loader, u []float64) float64 {
	// d := newDomain(ldr)
	d := newUniformDomain(ldr)
	b := d.newSubDomain(ldr.outlet)
	fmt.Printf("\n building sample HRUs and TOPMODEL\n\n")
	// smpl := b.toDefaultSample(topm, mann)
	smpl := b.toSampleU(u...)

	for _, c := range b.cids {
		if smpl.ws[c] == nil {
			log.Fatalln(" basin.Run() error: nil hru")
		}
	}
	b.printParam(u...)
	return b.evalCascKineWB(&smpl, true)
}

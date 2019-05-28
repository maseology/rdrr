package basin

import (
	"fmt"
	"log"
)

// Run a single simulation with water balance checking
func Run(ldr *Loader, rill, m, n float64) float64 {
	d := newDomain(ldr)
	b := d.newSubDomain(ldr.outlet)
	fmt.Printf("\n building sample HRUs and TOPMODEL\n\n")
	smpl := b.toDefaultSample(rill, m, n)
	for _, c := range b.cids {
		if smpl.ws[c] == nil {
			log.Fatalln(" basin.Run() error: nil hru")
		}
	}
	return b.evalCascKineWB(&smpl, true)
}

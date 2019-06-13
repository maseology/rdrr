package basin

import (
	"fmt"
	"log"
)

// Run a single simulation with water balance checking
func Run(ldr *Loader, rill, m, n float64) float64 {
	// d := newDomain(ldr)
	d := newUniformDomain(ldr)
	b := d.newSubDomain(ldr.outlet)
	fmt.Printf("\n building sample HRUs and TOPMODEL\n\n")
	// smpl := b.toDefaultSample(m, n)
	u := []float64{0.34934277975965045, 0.5066029698028864, 0.41250110165071685, 0.5997405837098619, 0.2536651978240927, 0.42335248701762207, 0.14393565861364455, 0.6010671573724449}
	smpl := b.toSampleU(u...)

	// smpl1 := b.toSampleU()
	// if ok, s := smpl.gw.IsSame(&smpl1.gw); !ok {
	// 	println("gw: " + s)
	// }

	for _, c := range b.cids {
		if smpl.ws[c] == nil {
			log.Fatalln(" basin.Run() error: nil hru")
		}
	}
	b.printParam(u...)
	return b.evalCascKineWB(&smpl, true)
}

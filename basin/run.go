package basin

import (
	"fmt"
	"log"
)

// // Run a single simulation with water balance checking
// func Run(ldr *Loader, u []float64) float64 {
// 	d := newUniformDomain(ldr)
// 	if d.frc.h.Nloc() != 1 && d.frc.h.LocationCode() <= 0 {
// 		log.Fatalf(" basin.Run error: unrecognized .met type\n")
// 	}
// 	b := d.newSubDomain(int(d.frc.h.Locations[0][0].(int32))) // gauge outlet id found in .met file
// 	fmt.Printf(" building sample HRUs and TOPMODEL\n\n")
// 	// smpl := b.toDefaultSample(topm, mann)
// 	smpl := b.toSampleU(u...)

// 	for _, c := range b.cids {
// 		if smpl.ws[c] == nil {
// 			log.Fatalln(" basin.Run() error: nil hru")
// 		}
// 	}
// 	b.printParam(u...)
// 	return b.evalCascWB(&smpl, 0., true)
// }

// RunDefault runs simulation with default parameters
func RunDefault(metfp string, topQo, topm, fcasc, freeboard float64, print bool) float64 {
	if masterDomain.IsEmpty() {
		log.Fatalf(" basin.RunDefault error: masterDomain is empty")
	}
	var b subdomain
	if len(metfp) == 0 && masterDomain.frc != nil {
		b = masterDomain.newSubDomain(masterForcing()) // gauge outlet cell id found in .met file
	} else {
		b = masterDomain.newSubDomain(loadForcing(metfp, print)) // gauge outlet cell id found in .met file
	}

	if print {
		fmt.Printf(" catchment area: %.1f km²\n", b.contarea/1000./1000.)
		fmt.Printf(" building sample HRUs and TOPMODEL\n\n")
	}

	smpl := b.toDefaultSample(topQo, topm, fcasc)
	for _, c := range b.cids {
		if smpl.ws[c] == nil {
			log.Fatalln(" basin.RunDefault() error: nil hru")
		}
	}

	if print {
		fmt.Printf(" running model..\n\n")
	}
	return b.evalCasc(&smpl, freeboard, print)
}

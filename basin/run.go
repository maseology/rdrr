package basin

import (
	"fmt"
	"log"
	"time"
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
func RunDefault(metfp string, topm, fcasc, freeboard float64, print bool) float64 {
	start := time.Now()
	if masterDomain.IsEmpty() {
		log.Fatalf(" basin.RunDefault error: masterDomain is empty\n")
	}
	var b subdomain
	if len(metfp) == 0 {
		if masterDomain.frc == nil {
			log.Fatalf(" basin.RunDefault error: no forcings made available\n")
		}
		b = masterDomain.newSubDomain(masterForcing()) // gauge outlet cell id found in .met file
	} else {
		b = masterDomain.newSubDomain(loadForcing(metfp, print)) // gauge outlet cell id found in .met file
	}

	if print {
		fmt.Printf(" sub-domain load complete %v\n", time.Now().Sub(start))
		fmt.Printf(" catchment area: %.1f km²\n", b.contarea/1000./1000.)
		fmt.Printf(" building sample HRUs and TOPMODEL\n")
		start = time.Now()
	}
	smpl := b.toDefaultSample(topm, fcasc)
	// for _, c := range b.cids {
	// 	if smpl.ws[c] == nil {
	// 		log.Fatalln(" basin.RunDefault() error: nil hru")
	// 	}
	// }

	if print {
		dir := "E:/ormgp_rdrr/check/"
		b.print(dir)
		smpl.print(dir)
		fmt.Printf(" sample load complete %v\n", time.Now().Sub(start))
		fmt.Printf(" number of subwatersheds: %d\n", len(smpl.gw))
		fmt.Printf("\n running model..\n\n")
	}

	return b.evalCascWB(&smpl, freeboard, print)
}

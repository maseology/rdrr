package basin

import (
	"fmt"
	"log"
	"github.com/maseology/mmio"
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
func RunDefault(metfp string, topm, fcasc, Qs, freeboard float64, print bool) float64 {
	tt := mmio.NewTimer()
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
		tt.Lap("sub-domain load complete")
		fmt.Printf(" catchment area: %.1f km²\n", b.contarea/1000./1000.)
		fmt.Printf(" building sample HRUs and TOPMODEL\n")		
	}
	smpl := b.toDefaultSample(topm, fcasc)
	// for _, c := range b.cids {
	// 	if smpl.ws[c] == nil {
	// 		log.Fatalln(" basin.RunDefault() error: nil hru")
	// 	}
	// }

	if print {
		tt.Lap("sample build complete")		
		dir := "S:/ormgp_rdrr/check/" //"E:/ormgp_rdrr/check/" //
		masterDomain.gd.SaveAs(dir + "masterDomain.gdef")
		b.print(dir)
		smpl.print(dir)
		tt.Lap("sample map printing")	
		fmt.Printf(" number of subwatersheds: %d\n", len(smpl.gw))
		fmt.Printf("\n running model..\n\n")
	}

	return b.evalCascWB(&smpl, Qs, freeboard, print)
}

package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/model"
)

func main() {

	const (
		// mdlPrfx = "S:/OWRC-RDRR/owrc."
		mdlPrfx = "M:/Peel/RDRR-PWRMM21/PWRMM21."          // "S:/Peel/PWRMM21."        //
		obsfp   = "M:/Peel/RDRR-PWRMM21/dat/obs/HY045.csv" // "S:/Peel/obs/02HB004.csv" //
		cid0    = 1340114                                  //                                    //
	)
	// 02HC033 1537675
	// HY045   1340114
	// 02HB004 2014386
	// 02HB008 1552736
	// 02HB024 1610724
	// single sws, little imperv

	fmt.Println("")
	tt := mmio.NewTimer()
	defer tt.Lap(fmt.Sprintf("\nRun complete. n processes: %v", runtime.GOMAXPROCS(0)))

	// load data
	dom := model.LoadDomain(mdlPrfx)
	tt.Print("Master Domain Load complete\n")
	model.DeleteMonitors(mdlPrfx+"out/", true) // also sets-up the output folder
	if err := dom.Frc.AddObservation(obsfp, dom.Strc.Acell, cid0); err != nil {
		log.Fatalln(err)
	}

	// run model
	TMQm, grdMin, kstrm, mcasc, soildepth, dinc, urbDiv := 4.770454, 0.02148, 0.973707, 0.445008, 0.663354, 0.343679, 0.207454
	ksat := []float64{5.97e-08, 1.07e-08, 1.28e-05, 7.61e-05, 0.002199092, 3.12e-06, 0.000228956, 7.51e-06}
	dom.RunSurfGeo(mdlPrfx+"out/", mdlPrfx+"check/", TMQm, grdMin, kstrm, mcasc, soildepth, dinc, urbDiv, ksat, cid0, true)

	// // sample models
	// model.PrepMC(mdlPrfx + "MC/")
	// dom.SampleSurfGeo(mdlPrfx, 1000, cid0)

	// // find optimal model
	// model.OptimizeDefault(nil, 1104986)
}

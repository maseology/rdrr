package main

import (
	"fmt"
	"runtime"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/model"
)

func main() {

	const (
		// mdlPrfx = "S:/OWRC-RDRR/owrc."
		mdlPrfx = "M:/Peel/RDRR-PWRMM21/PWRMM21."          // "S:/Peel/PWRMM21."        //
		obsfp   = "M:/Peel/RDRR-PWRMM21/dat/obs/HY045.csv" // "S:/Peel/obs/02HB004.csv" //
		cid0    = 1340114                                  //                                           //
	)
	// 02HC033 1537675
	// HY045   1340114
	// 02HB004 2014386
	// 02HB008 1552736
	// 02HB024 1610724

	fmt.Println("")
	tt := mmio.NewTimer()
	defer tt.Lap(fmt.Sprintf("\nRun complete. n processes: %v", runtime.GOMAXPROCS(0)))

	// load data
	dom := model.LoadDomain(mdlPrfx)
	tt.Print("Master Domain Load complete\n")
	model.DeleteMonitors(mdlPrfx+"out/", true) // also sets-up the output folder
	dom.Frc.AddObservation(obsfp, dom.Strc.Acell, cid0)

	// run model
	// TMQm, kstrm, mcasc, urbDiv, soildepth := 0.022384, 0.998708, 0.029728, 0.195728, 0.020582
	// ksat := []float64{1.10e-08, 1.18e-09, 2.13e-08, 1.43e-06, 0.000344009, 1.76e-08, 1.45e-09, 3.68e-06}
	kstrm, mcasc, urbDiv, soildepth := 0.996729, 0.015932, 0.5, 0.0354585
	TMQm := []float64{.3484373, .3484373}
	ksat := []float64{6.15e-08, 2.37e-06, 3.83e-06, 5.45e-05, 1.93e-05, 1.81e-08, 1.81e-09, 1.63e-08}
	dom.RunSurfGeo(mdlPrfx+"out/", mdlPrfx+"check/", kstrm, mcasc, soildepth, urbDiv, TMQm, ksat, cid0, true)

	// // sample models
	// model.PrepMC(mdlPrfx + "MC/")
	// dom.SampleSurfGeo(mdlPrfx, 2500, cid0)

	// // find optimal model
	// model.OptimizeDefault(nil, 1104986)
}

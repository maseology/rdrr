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
		mdlPrfx = "M:/Peel/RDRR-PWRMM21/PWRMM21."            // "S:/Peel/PWRMM21."        //
		obsfp   = "M:/Peel/RDRR-PWRMM21/dat/obs/02HC033.csv" // "S:/Peel/obs/02HB004.csv" //
		cid0    = 1537675                                    //                                    //
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
	// TMQm, grdMin, kstrm, mcasc, urbDiv, soildepth, dinc := 0.221871, 0.073682, 0.979411, 2.13048, 0.667674, 0.086067, 0.961614
	TMQm, grdMin, kstrm, mcasc, urbDiv, soildepth, dinc := 0.221, 0.05, 0.995, 2.13048, 0.9, 0.086067, 0.
	ksat := []float64{7.73e-09, 4.63e-06, 1.21e-06, 1.30e-05, 0.00577451, 4.92e-08, 0.006880688, 2.53e-08}
	dom.RunSurfGeo(mdlPrfx+"out/", mdlPrfx+"check/", TMQm, grdMin, kstrm, mcasc, soildepth, dinc, urbDiv, ksat, cid0, true)

	// // sample models
	// model.PrepMC(mdlPrfx + "MC/")
	// dom.SampleSurfGeo(mdlPrfx, 1000, cid0)

	// // find optimal model
	// model.OptimizeDefault(nil, 1104986)
}

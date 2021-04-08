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
		mdlPrfx = "M:/Peel/RDRR-PWRMM21/PWRMM21."            // "S:/Peel/PWRMM21."        //
		obsfp   = "M:/Peel/RDRR-PWRMM21/dat/obs/02HC033.csv" // "S:/Peel/obs/02HB004.csv" //
		cid0    = 1537675
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
	if err := dom.Frc.AddObservation(obsfp, dom.Strc.Acell, cid0); err != nil {
		log.Fatalln(err)
	}

	// run model
	TMQm, grdMin, kstrm, mcasc, soildepth, dinc, urbDiv := 0.803918, 0.000835, .995, 0.256054, 0.05, 1.489587, .9
	ksat := []float64{8.38e-9, 2.66e-09, 2.57e-07, 1.04e-05, 7.50e-05, 2.37e-07, 4.67e-08, 1.23e-07}
	fmt.Println(dom.RunSurfGeo(mdlPrfx+"out/", mdlPrfx+"check/", TMQm, grdMin, kstrm, mcasc, soildepth, dinc, urbDiv, ksat, cid0, true))

	// // sample models
	// model.PrepMC(mdlPrfx + "MC/")
	// dom.SampleSurfGeo(mdlPrfx, 1000, cid0)

	// // find optimal model
	// model.OptimizeDefault(nil, 1104986)
}

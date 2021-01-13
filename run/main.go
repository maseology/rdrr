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
		mdlPrfx = "S:/Peel/PWRMM21." // "M:/Peel/RDRR-PWRMM21/PWRMM21." //
		cid0    = 1552736
		obsfp   = "S:/Peel/obs/02HB008.csv" // "M:/Peel/RDRR-PWRMM21/dat/obs/02HB008.csv"
	)

	fmt.Println("")
	tt := mmio.NewTimer()
	defer tt.Lap(fmt.Sprintf("\nRun complete. n processes: %v", runtime.GOMAXPROCS(0)))

	// load data
	model.LoadMasterDomain(mdlPrfx)
	tt.Print("Master Domain Load complete\n")

	model.DeleteMonitors(mdlPrfx+"out/", true) // also sets-up the output folder
	if err := model.MasterDomain.Frc.AddObservation(obsfp, cid0); err != nil {
		log.Fatalln(err)
	}

	// // run model
	// TMQm := 6.7
	// grdMin := .5
	// kstrm := .995
	// mcasc := 3. // .001-10
	// soildepth := .815
	// kfact := .088
	// dinc := 8.5
	// fmt.Println(model.RunDefault(mdlPrfx+"out/", mdlPrfx+"check/", TMQm, grdMin, kstrm, mcasc, dinc, soildepth, kfact, cid0, true))

	// sample models
	model.PrepMC(mdlPrfx + "MC/")
	model.SampleMaster(mdlPrfx, 700, cid0)

	// // find optimal model
	// model.OptimizeDefault(nil, 1104986)
}

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
		mdlPrfx = "M:/Peel/RDRR-PWRMM21/PWRMM21." // "S:/Peel/PWRMM21."
		cid0    = 1552736
		obsfp   = "M:/Peel/RDRR-PWRMM21/dat/obs/02HB008.csv"
	)

	fmt.Println("")
	tt := mmio.NewTimer()
	defer tt.Lap(fmt.Sprintf("\nRun complete. n processes: %v", runtime.GOMAXPROCS(0)))

	// load data
	model.LoadMasterDomain(mdlPrfx)
	// if err := model.MasterDomain.Frc.AddObservation(obsfp, cid0); err != nil {
	// 	log.Fatalln(err)
	// }
	tt.Print("Master Domain Load complete\n")

	// run model
	model.DeleteMonitors(mdlPrfx+"out/", true) // also sets-up the output folder

	// // TMQm := 1.4380620030367803
	// // grng := 0.0019749463694001177
	// // soildepth := 0.8744999999999999
	// // kfact := 946.237161365793
	// TMQm := .01
	// grdMin := .0005
	// kstrm := .999
	// mcasc := .5 // .001-10
	// soildepth := 1.
	// kfact := 1.
	// dinc := 1.
	// fmt.Println(model.RunDefault(mdlPrfx+"out/", mdlPrfx+"check/", TMQm, grdMin, kstrm, mcasc, dinc, soildepth, kfact, cid0, true))

	// // find optimal model
	// model.OptimizeDefault(nil, 1104986)

	// sample models
	model.PrepMC(mdlPrfx + "MC/")
	model.SampleMaster(mdlPrfx, 3, cid0)

}

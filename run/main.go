package main

import (
	"fmt"
	"runtime"

	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/model"
)

func main() {

	// const (
	// 	mdlPrfx = "S:/OWRC-RDRR/owrc."
	// 	monFP   = "S:/OWRC-RDRR/owrc20-50-obs.obs"
	// )

	const (
		mdlPrfx = "M:/Peel/RDRR-PWRMM21/PWRMM21."
		monFP   = "M:/Peel/RDRR-PWRMM21/dat/elevation.real.uhdem.gauges_final.obs"
		obsFP   = "M:/Peel/RDRR-PWRMM21/dat/obs/02HB029.csv" // outlet=1750373
	)
	// const (
	// 	mdlPrfx = "S:/Peel/PWRMM21."
	// 	monFP   = "S:/Peel/elevation.real.uhdem.gauges_final.obs"
	// )
	// var obsFP = [...]string{"S:/Peel/1750373.obs"}

	fmt.Println("")
	tt := mmio.NewTimer()
	defer tt.Lap(fmt.Sprintf("\nRun complete. n processes: %v", runtime.GOMAXPROCS(0)))

	// load data
	model.LoadMasterDomain(mdlPrfx, monFP)
	tt.Print("Master Domain Load complete\n")

	// run model
	model.DeleteMonitors(mdlPrfx + "out/") // also sets-up the output folder
	// topm, smax, dinc, soildepth, kfact := .045394, .004987, .116692, .073995, 1.
	// topm, slpmx, dinc, soildepth, kfact, hmax := 0.01153, 2.287310, 0.104665, 1.435206, 33.153130, .01

	topm := .1
	slpx := .1
	dinc := 0.
	soildepth := 1.
	kfact := 1.
	hmax := 1.                                                                                                      // maximum mobile stor depth
	fmt.Println(model.RunDefault(mdlPrfx+"check/", obsFP, topm, hmax, slpx, dinc, soildepth, kfact, 1750373, true)) //897926

	// model.OptimizeDefault(nil, obsFP, 1750373)

	// // sample models
	// model.PrepMC(mdlPrfx + "MC/")
	// model.SampleMaster(mdlPrfx, 100)

}

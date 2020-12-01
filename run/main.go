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

	// const (
	// 	mdlPrfx = "S:/RDRR-02HJ005/02HJ005."
	// 	monFP   = "S:/RDRR-02HJ005/dat/02HJ005.obs"
	// 	obsFP   = "S:/RDRR-02HJ005/dat/02HJ005.csv"
	// 	outlet  = 757
	// )

	// const (
	// 	mdlPrfx = "M:/Peel/RDRR-PWRMM21/PWRMM21."
	// 	monFP   = "M:/Peel/RDRR-PWRMM21/dat/elevation.real.uhdem.gauges_final.obs"
	// 	obsFP   = "M:/Peel/RDRR-PWRMM21/dat/obs/02HB029.csv" // outlet=1750373
	// )
	const (
		mdlPrfx = "S:/Peel/PWRMM21."
		monFP   = "S:/Peel/elevation.real.uhdem.gauges_final.obs"
		obsFP   = "S:/Peel/02HB029.csv"
		outlet  = 1750373
	)

	fmt.Println("")
	tt := mmio.NewTimer()
	defer tt.Lap(fmt.Sprintf("\nRun complete. n processes: %v", runtime.GOMAXPROCS(0)))

	// load data
	model.LoadMasterDomain(mdlPrfx, monFP)
	tt.Print("Master Domain Load complete\n")

	// run model
	model.DeleteMonitors(mdlPrfx + "out/") // also sets-up the output folder

	// TMQm := 9.506832913616858
	// grng := 0.24545011902472752
	// soildepth := 0.6693802409889924
	// kfact := 15.405015002891941
	// model.RunDefault(mdlPrfx+"check/", obsFP, topm, hmax, slpx, dinc, soildepth, kfact, outlet, true)

	// sample model
	model.OptimizeDefault(nil, obsFP, outlet)

	// // sample models
	// model.PrepMC(mdlPrfx + "MC/")
	// model.SampleMaster(mdlPrfx, 200, outlet)

}

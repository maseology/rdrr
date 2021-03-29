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
		mdlPrfx = "M:/Peel/RDRR-PWRMM21/PWRMM21." //"S:/Peel/PWRMM21." //
		obsfp   = "S:/Peel/obs/02HB004.csv"       //"M:/Peel/RDRR-PWRMM21/dat/obs/02HB004.csv" // "S:/Peel/obs/02HB008.csv" //"M:/Peel/RDRR-PWRMM21/dat/obs/02HB008.csv" //
		cid0    = 2014386                         //-1                                         //  1552736
	)
	//02HB024

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

	// fobs := dom.Frc.O[0]
	// oobs, _ := postpro.GetObservations("C:/Users/Mason/Desktop/", "")
	// ooobs := oobs[1552736]
	// cobs, ii := make([]float64, len(fobs)), 0
	// dtb := time.Date(2010, 10, 1, 0, 0, 0, 0, time.UTC)
	// dte := time.Date(2020, 9, 30, 18, 0, 0, 0, time.UTC)
	// for i, t := range ooobs.T {
	// 	if t.Before(dtb) || t.After(dte) {
	// 		continue
	// 	}
	// 	cobs[ii] = ooobs.V[i]
	// 	ii++
	// }
	// _ = fobs
	// _ = cobs
	// fmt.Println("")

	// // run model
	// // // // TMQm := 1.
	// // // // grdMin := .01
	// // // // kstrm := .995
	// // // // mcasc := 1. // .001-10
	// // // // soildepth := .815
	// // // // kfact := 1.
	// // // // dinc := 10.5
	// // // TMQm := .633454
	// // // grdMin := .000379
	// // // kstrm := .954231
	// // // mcasc := .211821 // .001-10
	// // // soildepth := 1.298047
	// // // kfact := .002853
	// // // dinc := 1.114309
	// // TMQm := 3.87
	// // grdMin := .007215
	// // kstrm := .9959
	// // mcasc := .377441 // .001-10
	// // soildepth := .164
	// // kfact := .009681
	// // dinc := 1.5
	// // fmt.Println(dom.RunDefault(mdlPrfx+"out/", mdlPrfx+"check/", TMQm, grdMin, kstrm, mcasc, soildepth, kfact, dinc, cid0, true))
	// // // fmt.Println(model.RunDefault(mdlPrfx+"out/", mdlPrfx+"check/", 37.866772, 2.60e-05, 0.64884, 0.002168, 1.374418, 0.020174, 4.654649, 1552736, true))
	// TMQm, grdMin, kstrm, mcasc, soildepth, dinc := 0.208767, 0.000368, 0.985471, 0.235153, 0.4125, 0.264875
	// ksat := []float64{1.42e-07, 4.00e-09, 1.06e-07, 3.43e-05, 0.006156758, 1.23e-09, 4.64e-05, 2.27e-08}
	// fmt.Println(dom.RunSurfGeo(mdlPrfx+"out/", mdlPrfx+"check/", TMQm, grdMin, kstrm, mcasc, soildepth, dinc, ksat, cid0, true))

	// sample models
	model.PrepMC(mdlPrfx + "MC/")
	dom.SampleSurfGeo(mdlPrfx, 1000, cid0)

	// // find optimal model
	// model.OptimizeDefault(nil, 1104986)
}

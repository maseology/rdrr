package main

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"time"

	"rdrr/model"
	"rdrr/opt"

	"github.com/maseology/glbopt"
	"github.com/maseology/mmio"
	"github.com/maseology/objfunc"
	mrg63k3a "github.com/maseology/pnrg/MRG63k3a"
)

// var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

const (
	mdlPrfx           = "S:/Peel/PWRMM21." // "M:/Peel/RDRR-PWRMM21/PWRMM21."
	annualavgRecharge = 150.               // [mmpyr] -- for initial conditions

	checkmode = false
	optimize  = true
)

func main() {
	// flag.Parse()
	// if *cpuprofile != "" {
	// 	f, err := os.Create(*cpuprofile)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	pprof.StartCPUProfile(f)
	// 	defer pprof.StopCPUProfile()
	// }

	fmt.Println("")
	tt := mmio.NewTimer()
	defer tt.Lap(fmt.Sprintf("\nRun complete. n processes: %v", runtime.GOMAXPROCS(0)))

	// load domain
	dom := model.LoadDomain(mdlPrfx)
	cidout := func() int {
		for i, c := range dom.Obs.Oqxr {
			if c == dom.Strc.CID0 {
				return i
			}
		}
		return -1
	}()
	domeval := dom.EvaluateVerbose

	tt.Print("Master Domain Load complete")
	dom.Print()

	evaluate := func(TOPMODELm, acasc, maxFcasc, soildepth, dinc float64, prnt bool) float64 {

		// initialize hydrological elements with parameter assignment
		lus, xg, xm, cxr, gxr := dom.Parameterize(acasc, soildepth, maxFcasc, dinc, TOPMODELm, prnt)
		dms := dom.FindDm0s(lus, annualavgRecharge, cxr, xg, prnt)

		// check loaded data
		if checkmode {
			dom.PreRunCheck(lus, cxr, xg, xm)
			os.Exit(22)
		}

		// run model
		hyd0 := domeval(lus, dms, xg, xm, gxr, prnt) // [m³/s]

		return 1. - objfunc.NSEsmooth(dom.Obs.Oq[cidout], dom.Obs.ToDaily(dom.Frc.T, hyd0), 3)
	}

	txtfmt := "parameters:\n\tTMQm=\t\t%v\n\tacasc=\t\t%v\n\tmaxcasc=\t%v\n\tsoildepth=\t%v\n\tdinc=\t\t%v\n\n"
	if optimize {
		// optimization
		gen := func(u []float64) float64 {
			TOPMODELm, acasc, maxFcasc, soildepth, dinc := opt.Par5(u)
			return evaluate(TOPMODELm, acasc, maxFcasc, soildepth, dinc, false)
		}
		rng := rand.New(mrg63k3a.New())
		rng.Seed(time.Now().UnixNano())

		fmt.Println(" optimizing..")
		uFinal, _ := glbopt.SCE(8, 5, rng, gen, true)

		TOPMODELm, acasc, maxFcasc, soildepth, dinc := opt.Par5(uFinal)
		fmt.Printf("\nfinal "+txtfmt, TOPMODELm, acasc, maxFcasc, soildepth, dinc)
		fmt.Println(evaluate(TOPMODELm, acasc, maxFcasc, soildepth, dinc, true), []float64{acasc})

	} else {

		TOPMODELm, acasc, maxFcasc, soildepth, dinc := .2657, 0.01, .932, 1.5, -.514
		// TOPMODELm, acasc, maxFcasc, soildepth, dinc := opt.Par5([]float64{0.027, 0.999, 0.662, 0.998, 0.243})
		fmt.Printf(txtfmt, TOPMODELm, acasc, maxFcasc, soildepth, dinc)
		fmt.Println(evaluate(TOPMODELm, acasc, maxFcasc, soildepth, dinc, true))

	}
}

package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/maseology/glbopt"
	"github.com/maseology/mmaths"
	"github.com/maseology/objfunc"
	mrg63k3a "github.com/maseology/pnrg/MRG63k3a"

	"github.com/maseology/goHydro/met"
	"github.com/maseology/goHydro/pet"
	"github.com/maseology/goHydro/solirrad"
	"github.com/maseology/mmio"
)

func main() {
	// INPUTS
	fp := "E:/climate_data/PanET/calibration/6153300_merged.met"
	lat := 43.28

	tt := mmio.NewTimer()
	defer tt.Print("complete!")

	si := solirrad.New(lat, 0., 0.)
	hdr, dat, err := met.ReadMET(fp, true)
	if err != nil {
		log.Fatalf("%v", err)
	}

	dat.Print(hdr.WBlist())
	x := hdr.WBDCxr()
	dts, obs := dat.Get(x["Evaporation"])
	_, tx := dat.Get(x["MaxDailyT"])
	_, tn := dat.Get(x["MinDailyT"])
	// _, r := dat.Get(x["Rainfall"])
	// _, s := dat.Get(x["Snowfall"])

	gen := func(a, b, t, g, alpha, beta float64) (sim, mx, mobs, msim []float64) {
		etRadToGlobal := func(Ke, a, b, g, tx, tn float64) float64 {
			// see pg 151 in DeWalle & Rango; attributed to Bristow and Campbell (1984)
			// ref: Bristow, K.L. and G.S. Campbell, 1984. On the relationship between incoming solar radiation and daily maximum and minimum temperature. Agricultural and Forest Meteorology 31(2):159--166.
			return Ke * a * (1. - math.Exp(-b*math.Pow(tx-tn, g)))
		}
		// etRadToGlobal := func(Ke, a, b, nN float64) float64 {
		// 	// the Prescott-type equation (Novák, 2012, pg.232)
		// 	return Ke * (a + b*nN)
		// }
		makkink := func(tx, tn, alpha, beta, a, b, g float64, doy int) float64 {
			tm := (tx + tn) / 2.
			Kg := etRadToGlobal(si.PSIdaily(doy), a, b, g, tx, tn)
			return pet.Makkink(Kg, tm, 101300., alpha, beta)
		}

		xr, mc := make([]int, len(dts)), -1
		for i, dt := range dts {
			if dt.Day() == 1 {
				mc++
			}
			xr[i] = mc
		}
		mx, mobs, msim = make([]float64, mc+1), make([]float64, mc+1), make([]float64, mc+1)
		sim = make([]float64, len(dts))
		for i, dt := range dts {
			// nN := 1. // ratio of sunshine hours (n) to total possible ( N = si.DaylightHours(doy) )
			// if r[i]+s[i] > t {
			// 	nN = g
			// }
			// sim[i] = makkink(tx[i], tn[i], alpha, beta, nN, dt.YearDay())
			sim[i] = makkink(tx[i], tn[i], alpha, beta, a, b, g, dt.YearDay())
			mobs[xr[i]] += obs[i]
			msim[xr[i]] += sim[i]
			mx[xr[i]] = float64(xr[i])
		}
		return
	}

	fmt.Println(" optimizing..")
	smple := func(u []float64) (a, b, t, g, alpha, beta float64) {
		a = 1.
		b = mmaths.LinearTransform(0., 1., u[0])
		t = 0.
		g = mmaths.LinearTransform(0., 2., u[1])
		alpha = mmaths.LinearTransform(0., 2., u[2])
		beta = mmaths.LinearTransform(-.005, 0.005, u[3])
		// a = mmaths.LinearTransform(0., 1., u[0])
		// b = mmaths.LinearTransform(0., 1., u[1])
		// t = mmaths.LinearTransform(0., 2., u[2])
		// g = mmaths.LinearTransform(0., 1., u[3])
		// alpha = mmaths.LinearTransform(0., 2., u[4])
		// beta = mmaths.LinearTransform(-.005, 0.005, u[5])
		return
	}
	eval := func(u []float64) float64 {
		a, b, t, g, alpha, beta := smple(u)
		_, _, mobs, msim := gen(a, b, t, g, alpha, beta)
		return objfunc.NSE(mobs, msim)
	}
	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	uFinal, _ := glbopt.SCE(200, 4, rng, eval, false)
	aFinal, bFinal, tFinal, gFinal, alphaFinal, betaFinal := smple(uFinal)

	// aFinal, bFinal, tFinal, gFinal, alphaFinal, betaFinal := 0.3750253781645832, 0.6862718561400876, 0.0007986224334156789, 0.2732214983494662, 0.6783274265762209, -0.0009731523799474152
	// aFinal, bFinal, tFinal, gFinal, alphaFinal, betaFinal := 0.37503, 0.68627, 0.0007986, 0.2732, 0.6783, -0.00097315
	sim, mx, mobs, msim := gen(aFinal, bFinal, tFinal, gFinal, alphaFinal, betaFinal)
	fmt.Println(aFinal, bFinal, tFinal, gFinal, alphaFinal, betaFinal)
	fmt.Println(" monthly NSE: ", objfunc.NSE(mobs, msim))
	mmio.Temporal("t.png", dts, map[string][]float64{"PanET": obs, "simulated": sim}, 48.)
	mmio.Line("m.png", mx, map[string][]float64{"obs": mobs, "sim": msim}, 36.)
	mmio.Scatter11("s.png", mobs, msim)
}

/*
 --using Bristow and Campbell
1 0.06142000937166982 0 0.8993576921076015 1.3077260971977016 -0.0003611220703204068
 monthly NSE:  0.8240671569965909

 --using Prescott
0.3750253781645832 0.6862718561400876 0.0007986224334156789 0.2732214983494662 0.6783274265762209 -0.0009731523799474152
 monthly NSE:  0.8342736213208906

0.36391264760553416 0.6178035871430676 0.0008106189343911625 ??? 0.7582615752492149 -0.0009258148957828607
 monthly NSE:  0.8323671875029506
0.32519046532291923 0.5488390650905296 0.0023927531012973456 0.0919485460496158 0.7911638047329248 -0.0010166328127383774
 monthly NSE:  0.8320198682931542
*/

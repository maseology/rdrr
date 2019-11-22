package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/maseology/glbopt"
	"github.com/maseology/mmaths"
	"github.com/maseology/objfunc"
	mrg63k3a "github.com/maseology/pnrg/MRG63k3a"

	"github.com/maseology/goHydro/snowpack"
	"github.com/maseology/mmio"
)

func main() {
	tt := mmio.NewTimer()
	defer tt.Print("snowmelt evaluation complete")
	fp := "S:/ormgp_rdrr/met/stations.json" // "M:/ORMGP/met/stations.json" // from pyMet align.py 191112

	dat := func() [][]float64 { // load data
		f, err := os.Open(fp)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		b, err := ioutil.ReadAll(f)
		if err != nil {
			panic(err)
		}

		var dat [][]float64
		if err := json.Unmarshal(b, &dat); err != nil {
			panic(err)
		}
		sr, ss, cnt := 0., 0., 0.
		for _, v := range dat {
			if len(v) != 5 { // ['MaxDailyT', 'MinDailyT', 'Rainfall', 'Snowfall', 'Snowdepth']
				panic("not a valid input size")
			}
			sr += v[2]
			ss += v[3]
			cnt++
		}
		fmt.Printf("\n -- total rain: %.1f, total snow: %.1f (mm/yr)\n", sr/cnt*365.24, ss/cnt*365.24)
		return dat
	}()

	gen := func(tindex, ddfc, baseT, tsf float64) (x, obs, sim []float64) {
		sp := snowpack.NewCCF(tindex, 0.0045, ddfc, baseT, tsf)
		x, obs, sim = make([]float64, len(dat)), make([]float64, len(dat)), make([]float64, len(dat))
		for k, v := range dat { // ['MaxDailyT', 'MinDailyT', 'Rainfall', 'Snowfall', 'Snowdepth']
			x[k] = float64(k)
			obs[k] = v[4] / 100. // convert from cm depth
			tm := (v[0] + v[1]) / 2.
			sp.Update(v[2]/1000., v[3]/1000., tm)
			_, d := sp.Properties()
			sim[k] = d
			// fmt.Println(k, v, m, d)
		}
		return
	}

	smple := func(u []float64) (tindex, ddfc, baseT, tsf float64) {
		tindex = mmaths.LogLinearTransform(0.0002, 0.05, u[0]) // CCF temperature index; range .0002 to 0.0005 m/°C/d -- roughly 1/10 DDF (pg.278)
		ddfc = mmaths.LinearTransform(0., 2.5, u[1])           // DDF adjustment factor based on pack density, see DeWalle and Rango, pg. 275; Ref: Martinec (1960)=1.1
		baseT = mmaths.LinearTransform(-5., 5., u[2])          // base/critical temperature (°C)
		tsf = mmaths.LinearTransform(0.1, 0.6, u[3])           // TSF (surface temperature factor), 0.1-0.5 have been used
		// ddf = mmaths.LinearTransform(0.001, 0.008, u[1])           // (initial) degree-day/melt factor; range .001 to .008 m/°C/d  (pg.275)
		return
	}
	ofnc := func(obs, sim []float64) float64 {
		obs0, sim0 := []float64{}, []float64{}
		for i := 0; i < len(obs); i++ {
			if obs[i] == 0. && sim[i] == 0. {
				continue
			}
			obs0 = append(obs0, obs[i])
			sim0 = append(sim0, sim[i])
		}
		return objfunc.NSE(obs0, sim0)
	}
	eval := func(u []float64) float64 {
		tindex, ddfc, baseT, tsf := smple(u)
		_, obs, sim := gen(tindex, ddfc, baseT, tsf)
		return ofnc(obs, sim)
	}
	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	uFinal, _ := glbopt.SCE(200, 4, rng, eval, false)

	func(u []float64) { // print outputs
		tindex, ddfc, baseT, tsf := smple(u) // 0.009981, 1.794442, -2.035386, 0.211562
		fmt.Printf("\n tindex, ddfc, baseT, tsf := %f, %f, %f, %f\n", tindex, ddfc, baseT, tsf)
		_, obs, sim := gen(tindex, ddfc, baseT, tsf)
		// mmio.Line("t.png", x, map[string][]float64{"obs": obs, "sim": sim}, 128.)
		// mmio.Scatter11("s.png", obs, sim)

		iobs, isim, c := make([]interface{}, len(obs)), make([]interface{}, len(obs)), make([]interface{}, len(obs))
		for i := 0; i < len(obs); i++ {
			iobs[i] = obs[i]
			isim[i] = sim[i]
			c[i] = i
		}
		mmio.WriteCSV("t.csv", "d,obs,sim", c, iobs, isim)
		fmt.Println(" NSE: ", ofnc(obs, sim))
	}(uFinal)
}

/*
 tindex, ddfc, baseT, tsf := 0.009981, 1.794442, -2.035386, 0.211562
 NSE:  0.340576674802623
*/

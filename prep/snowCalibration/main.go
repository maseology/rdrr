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

	dat := func(fp string) [][]float64 { // load data
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
		for _, v := range dat {
			if len(v) != 5 { // ['MaxDailyT', 'MinDailyT', 'Rainfall', 'Snowfall', 'Snowdepth']
				panic("not a valid input size")
			}
		}
		return dat
	}("M:/ORMGP/met/stations.json")

	gen := func(tindex, ddf, ddfc, baseT, tsf float64) (x, obs, sim []float64) {
		sp := snowpack.NewCCF(tindex, ddf, ddfc, baseT, tsf)
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

	smple := func(u []float64) (tindex, ddf, ddfc, baseT, tsf float64) {
		tindex = mmaths.LinearTransform(0.0002, 0.0005, u[0])
		ddf = mmaths.LinearTransform(0.001, 0.008, u[1])
		ddfc = mmaths.LinearTransform(0.85, 1.5, u[2])
		baseT = mmaths.LinearTransform(-5., 5., u[3])
		tsf = mmaths.LinearTransform(0.1, 0.6, u[4])
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
		tindex, ddf, ddfc, baseT, tsf := smple(u)
		_, obs, sim := gen(tindex, ddf, ddfc, baseT, tsf)
		return ofnc(obs, sim)
	}
	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	uFinal, _ := glbopt.SCE(200, 5, rng, eval, false)

	func(u []float64) { // print outputs
		tindex, ddf, ddfc, baseT, tsf := smple(u)
		_, obs, sim := gen(tindex, ddf, ddfc, baseT, tsf)
		// mmio.Line("t.png", x, map[string][]float64{"obs": obs, "sim": sim}, 128.)
		// mmio.Scatter11("s.png", obs, sim)

		iobs, isim, c := make([]interface{}, len(obs)), make([]interface{}, len(obs)), make([]interface{}, len(obs))
		for i := 0; i < len(obs); i++ {
			iobs[i] = obs[i]
			isim[i] = sim[i]
			c[i] = i
		}
		mmio.WriteCSV("t.csv", "d,obs,sim", c, iobs, isim)
		fmt.Println(" monthly NSE: ", objfunc.NSE(obs, sim))
	}(uFinal)
}

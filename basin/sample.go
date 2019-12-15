package basin

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"runtime"
	"time"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
	"github.com/maseology/mmio"
	"github.com/maseology/montecarlo"
	"github.com/maseology/montecarlo/smpln"
	mrg63k3a "github.com/maseology/pnrg/MRG63k3a"
)

type sample struct {
	ws hru.WtrShd        // hru watershed
	gw map[int]*gwru.TMQ // topmodel
	p0 map[int]float64
	// swsr, celr, p0, p1 map[int]float64
}

func (s *sample) copy() sample {
	return sample{
		ws: hru.CopyWtrShd(s.ws),
		p0: mmio.CopyMapif(s.p0),
		gw: func(origTMQ map[int]*gwru.TMQ) map[int]*gwru.TMQ {
			newTMQ := make(map[int]*gwru.TMQ, len(origTMQ))
			for k, v := range origTMQ {
				cpy := v.Copy()
				newTMQ[k] = &cpy
			}
			return newTMQ
		}(s.gw),
	}
}

func (s *sample) print(dir string) error {
	mmio.WriteRMAP(dir+"s.p0.rmap", s.p0, false)
	mmio.DeleteFile(dir + "s.gw.Qs.rmap")
	mmio.DeleteFile(dir + "s.gw.g-ti.rmap")
	for _, v := range s.gw {
		mmio.WriteRMAP(dir+"s.gw.Qs.rmap", v.Qs, true)
		mmio.WriteRMAP(dir+"s.gw.g-ti.rmap", v.RelTi(), true)
	}
	perc, fimp, smacap, srfcap := make(map[int]float64, len(s.ws)), make(map[int]float64, len(s.ws)), make(map[int]float64, len(s.ws)), make(map[int]float64, len(s.ws))
	for c, h := range s.ws {
		perc[c], fimp[c], smacap[c], srfcap[c] = h.PercFimpCap()
	}
	mmio.WriteRMAP(dir+"s.ws.perc.rmap", perc, false)
	mmio.WriteRMAP(dir+"s.ws.fimp.rmap", fimp, false)
	mmio.WriteRMAP(dir+"s.ws.smacap.rmap", smacap, false)
	mmio.WriteRMAP(dir+"s.ws.srfcap.rmap", srfcap, false)
	return nil
}

// SampleDefault samples a default-parameter model to a given basin outlet
func SampleDefault(metfp, outprfx string, nsmpl int) {
	b, ok := masterToSubomain(metfp)
	if !ok {
		return
	}
	fmt.Printf(" catchment area: %.1f km²\n\n", b.contarea/1000./1000.)
	dt, y, ep, obs, intvl, nstep := b.getForcings()
	v := func() float64 {
		// h2cms := b.contarea / b.frc.h.IntervalSec() // [m/ts] to [m³/s] conversion factor
		m, n, c := 0., 0., 0.
		for i := range obs {
			if !math.IsNaN(obs[i]) {
				m += obs[i] //* h2cms
				c++
			}
		}
		m /= c // mean
		for i := range obs {
			if !math.IsNaN(obs[i]) {
				// n += math.Pow(obs[i]*h2cms-m, 2.)
				n += math.Pow(obs[i]-m, 2.)
			}
		}
		return n / c // population variance
	}()

	gen := func(u []float64) float64 {
		m, smax, dinc, soildepth, kfact := par5(u)
		smpl := b.toDefaultSample(m, smax, soildepth, kfact)
		fmt.Print(".")
		return b.eval(&smpl, dt, y, ep, obs, intvl, nstep, dinc, m, false)
	}

	tt := mmio.NewTimer()
	u, f, d := montecarlo.RankedUnBiased(gen, nSmplDim, nsmpl)

	tt.Lap("\nsampling complete")
	t, err := mmio.NewTXTwriter(outprfx + "_smpl.csv")
	if err != nil {
		log.Fatalf("SampleDefault %s save error: %v", outprfx+"_smpl.csv", err)
	}
	t.WriteLine(fmt.Sprintf("rank(of %d),eval,m,smax,dinc,soildepth,kfact", nsmpl))
	for i, dd := range d {
		nse := math.Max(1.-math.Pow(f[dd], 2.)/v, -4.) // converting to nash-sutcliffe
		m, smax, dinc, soildepth, kfact := par5(u[dd])
		t.WriteLine(fmt.Sprintf("%d,%f,%f,%f,%f,%f,%f", i+1, nse, m, smax, dinc, soildepth, kfact))
	}
	t.Close()
	runtime.GC()
	tt.Lap("results save complete")
}

// SampleMaster samples a default-parameter full-domain model
func SampleMaster(outdir string, nsmpl int) {
	if masterDomain.IsEmpty() {
		log.Fatalf(" basin.RunDefault error: masterDomain is empty")
	}
	fmt.Println("Building Sub Domain..")
	var b subdomain
	if masterDomain.frc == nil {
		log.Fatalf(" basin.RunMaster error: no forcings made available\n")
	}
	frc, _ := masterForcing()
	b = masterDomain.noSubDomain(frc)
	b.mdldir = outdir
	b.cid0 = -1
	dt, y, ep, obs, intvl, nstep := b.getForcings()
	if len(b.rtr.swscidxr) == 1 {
		b.rtr.swscidxr = map[int][]int{-1: b.cids}
		b.rtr.sws = make(map[int]int, b.ncid)
		for _, c := range b.cids {
			b.rtr.sws[c] = -1
		}
	}
	// if len(metfp) == 0 {
	// 	if masterDomain.frc == nil {
	// 		log.Fatalf(" basin.RunDefault error: no forcings made available\n")
	// 	}
	// 	b = masterDomain.newSubDomain(masterForcing()) // gauge outlet cell id found in .met file
	// } else {
	// 	b = masterDomain.newSubDomain(loadForcing(metfp, true)) // gauge outlet cell id found in .met file
	// }
	fmt.Printf(" catchment area: %.1f km²\n\n", b.contarea/1000./1000.)

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	sp := smpln.NewLHC(rng, nsmpl, nSmplDim, false)

	printParams := func(m, smax, dinc, soildepth, kfact float64) {
		tw, err := mmio.NewTXTwriter(mondir + "params.txt")
		defer tw.Close()
		if err != nil {
			log.Fatalf("SampleMaster error: %v", err)
		}
		tw.WriteLine(mmio.MMtime(time.Now()))
		tw.WriteLine(mondir)
		tw.WriteLine(fmt.Sprintf("m\t%f", m))
		tw.WriteLine(fmt.Sprintf("smax\t%f", smax))
		tw.WriteLine(fmt.Sprintf("dinc\t%f", dinc))
		tw.WriteLine(fmt.Sprintf("soildepth\t%f", soildepth))
		tw.WriteLine(fmt.Sprintf("kfact\t%f", kfact))
	}

	gen := func(u []float64) {
		setMCdir()
		m, smax, dinc, soildepth, kfact := par5(u)
		go printParams(m, smax, dinc, soildepth, kfact)
		smpl := b.toDefaultSample(m, smax, soildepth, kfact)
		b.eval(&smpl, dt, y, ep, obs, intvl, nstep, dinc, m, false)
		WaitMonitors()
		compressMC(masterDomain.gd)
	}

	tt := mmio.NewTimer()
	fmt.Printf(" running %d samples from %d dimensions..\n", nsmpl, nSmplDim)

	for k := 0; k < nsmpl; k++ {
		ut := make([]float64, nSmplDim)
		for j := 0; j < nSmplDim; j++ {
			ut[j] = sp.U[j][k]
		}
		gen(ut)
		fmt.Print(".")
	}

	tt.Lap("\nsampling complete")
}

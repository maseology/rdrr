package basin

import (
	"fmt"
	"log"
	"math"
	"math/rand"
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
func SampleDefault(metfp, outdir string, nsmpl int) {
	if masterDomain.IsEmpty() {
		log.Fatalf(" basin.RunDefault error: masterDomain is empty")
	}
	var b subdomain
	if len(metfp) == 0 {
		if masterDomain.frc == nil {
			log.Fatalf(" basin.RunDefault error: no forcings made available\n")
		}
		b = masterDomain.newSubDomain(masterForcing()) // gauge outlet cell id found in .met file
	} else {
		b = masterDomain.newSubDomain(loadForcing(metfp, true)) // gauge outlet cell id found in .met file
	}
	fmt.Printf(" catchment area: %.1f km²\n\n", b.contarea/1000./1000.)

	gen := func(u []float64) float64 {
		m, smax, dinc, soildepth, kfact := par5(u)
		smpl := b.toDefaultSample(m, smax, soildepth, kfact)
		return b.eval(&smpl, dinc, m, false)
	}

	tt := mmio.NewTimer()
	fmt.Printf(" running %d samples from %d dimensions..\n", nsmpl, nSmplDim)
	u, f, d := montecarlo.RankedUnBiased(gen, nSmplDim, nsmpl)

	v := func() float64 {
		h2cms := b.contarea / b.frc.h.IntervalSec() // [m/ts] to [m³/s] conversion factor
		o, i, x := make([]float64, b.frc.h.Nstep()), 0, b.frc.h.WBDCxr()
		if xj, ok := x["UnitDischarge"]; ok {
			for k := range b.frc.c.T {
				o[i] = b.frc.c.D[k][0][xj] * h2cms
				i++
			}
			m, n, c := 0., 0., 0.
			for i := range o {
				if !math.IsNaN(o[i]) {
					m += o[i]
					c++
				}
			}
			m /= c // mean
			for i := range o {
				if !math.IsNaN(o[i]) {
					n += math.Pow(o[i]-m, 2.)
				}
			}
			return n / c // population variance
		}
		fmt.Println(" SampleDefault error, no UnitDischarge available")
		return math.NaN()
	}()

	tt.Lap("sampling complete")
	t, err := mmio.NewTXTwriter(outdir + "sample.csv")
	defer t.Close()
	if err != nil {
		log.Fatalf("SampleDefault sample.csv save error: %v", err)
	}
	t.WriteLine(fmt.Sprintf("rank(of %d),eval,m,smax,dinc,soildepth,kfact", nsmpl))
	for i, dd := range d {
		nse := 1. - math.Pow(f[dd], 2.)/v // converting to nash-sutcliffe
		m, smax, dinc, soildepth, kfact := par5(u[dd])
		t.WriteLine(fmt.Sprintf("%d,%f,%f,%f,%f,%f,%f", i+1, nse, m, smax, dinc, soildepth, kfact))
	}
	tt.Lap("results save complete")
}

// SampleMaster samples a default-parameter full-domain model
func SampleMaster(outdir string, nsmpl int) {
	if masterDomain.IsEmpty() {
		log.Fatalf(" basin.RunDefault error: masterDomain is empty")
	}
	var b subdomain
	if masterDomain.frc == nil {
		log.Fatalf(" basin.RunMaster error: no forcings made available\n")
	}
	frc, _ := masterForcing()
	b = masterDomain.newSubDomain(frc, -1)
	b.mdldir = outdir
	b.cid0 = -1
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
		b.eval(&smpl, dinc, m, false)
		WaitMonitors()
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

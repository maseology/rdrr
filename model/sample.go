package model

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
	"github.com/maseology/mmio"
	"github.com/maseology/montecarlo/smpln"
	mrg63k3a "github.com/maseology/pnrg/MRG63k3a"
)

// sample is a parameterized subdomain
type sample struct {
	ws    hru.WtrShd        // hru watershed
	gw    map[int]*gwru.TMQ // topmodel
	cascf map[int]float64   // cascade fraction
	dir   string
	// swsr, celr, p0, p1 map[int]float64
}

// func (s *sample) copy() sample {
// 	return sample{
// 		ws:    hru.CopyWtrShd(s.ws),
// 		cascf: mmio.CopyMapif(s.cascf),
// 		gw: func(origTMQ map[int]*gwru.TMQ) map[int]*gwru.TMQ {
// 			newTMQ := make(map[int]*gwru.TMQ, len(origTMQ))
// 			for k, v := range origTMQ {
// 				cpy := v.Copy()
// 				newTMQ[k] = &cpy
// 			}
// 			return newTMQ
// 		}(s.gw),
// 	}
// }

func (s *sample) write(dir string) error {
	mmio.WriteRMAP(dir+"s.cascf.rmap", s.cascf, false)
	mmio.DeleteFile(dir + "s.gw.drel.rmap")
	mmio.DeleteFile(dir + "s.gw.Qs.rmap")
	// mmio.DeleteFile(dir + "s.gw.g-ti.rmap")
	for _, v := range s.gw {
		mmio.WriteRMAP(dir+"s.gw.drel.rmap", v.D, true)
		mmio.WriteRMAP(dir+"s.gw.Qs.rmap", v.Qs, true)
		// mmio.WriteRMAP(dir+"s.gw.g-ti.rmap", v.RelTi(), true) // = drel/m
	}
	perc, fimp, smacap, srfcap := make(map[int]float64, len(s.ws)), make(map[int]float64, len(s.ws)), make(map[int]float64, len(s.ws)), make(map[int]float64, len(s.ws))
	for c, h := range s.ws {
		perc[c], fimp[c], smacap[c], srfcap[c] = h.Perc, h.Fimp, h.Sma.Cap, h.Sdet.Cap
	}
	mmio.WriteRMAP(dir+"s.ws.perc.rmap", perc, false)
	mmio.WriteRMAP(dir+"s.ws.fimp.rmap", fimp, false)
	mmio.WriteRMAP(dir+"s.ws.smacap.rmap", smacap, false)
	mmio.WriteRMAP(dir+"s.ws.Sdetcap.rmap", srfcap, false)
	return nil
}

// // SampleDefault samples a default-parameter model to a given basin outlet
// func SampleDefault(metfp, outprfx string, nsmpl int) {
// 	b := masterDomain.newSubDomain(masterDomain.frc, -1)
// 	fmt.Printf(" catchment area: %.1f km²\n\n", b.contarea/1000./1000.)
// 	// dt, y, ep, obs, intvl, nstep := b.getForcings()
// 	obs := []float64{} // as was in  b.getForcings()
// 	v := func() float64 {
// 		// h2cms := b.contarea / b.frc.h.IntervalSec() // [m/ts] to [m³/s] conversion factor
// 		m, n, c := 0., 0., 0.
// 		for i := range obs {
// 			if !math.IsNaN(obs[i]) {
// 				m += obs[i] //* h2cms
// 				c++
// 			}
// 		}
// 		m /= c // mean
// 		for i := range obs {
// 			if !math.IsNaN(obs[i]) {
// 				// n += math.Pow(obs[i]*h2cms-m, 2.)
// 				n += math.Pow(obs[i]-m, 2.)
// 			}
// 		}
// 		return n / c // population variance
// 	}()

// 	gen := func(u []float64) float64 {
// 		// m, hmax, smax, dinc, soildepth, kfact := par6(u)
// 		m, grng, soildepth, kfact := par4(u)
// 		smpl := b.toDefaultSample(m, grng, soildepth, kfact)
// 		fmt.Print(".")
// 		return b.evaluate(&smpl, 0., m, false)
// 	}

// 	tt := mmio.NewTimer()
// 	u, f, d := montecarlo.RankedUnBiased(gen, nSmplDim, nsmpl)

// 	tt.Lap("\nsampling complete")
// 	t, err := mmio.NewTXTwriter(outprfx + "_smpl.csv")
// 	if err != nil {
// 		log.Fatalf("SampleDefault %s save error: %v", outprfx+"_smpl.csv", err)
// 	}
// 	t.WriteLine(fmt.Sprintf("rank(of %d),eval,m,smax,dinc,soildepth,kfact", nsmpl))
// 	for i, dd := range d {
// 		nse := math.Max(1.-math.Pow(f[dd], 2.)/v, -4.) // converting to nash-sutcliffe
// 		// m, hmax, smax, dinc, soildepth, kfact := par6(u[dd])
// 		// t.WriteLine(fmt.Sprintf("%d,%f,%f,%f,%f,%f,%f,%f", i+1, nse, m, hmax, smax, dinc, soildepth, kfact))
// 		m, grng, soildepth, kfact := par4(u[dd])
// 		t.WriteLine(fmt.Sprintf("%d,%f,%f,%f,%f,%f", i+1, nse, m, grng, soildepth, kfact))
// 	}
// 	t.Close()
// 	runtime.GC()
// 	tt.Lap("results save complete")
// }

// SampleMaster samples a default-parameter full-domain model
func SampleMaster(outdir string, nsmpl, outlet int) {
	if MasterDomain.IsEmpty() {
		log.Fatalf(" basin.RunDefault error: MasterDomain is empty")
	}
	fmt.Println("Building Sub Domain..")
	var b subdomain
	if MasterDomain.Frc == nil {
		log.Fatalf(" basin.RunMaster error: no forcings made available\n")
	}

	b = MasterDomain.newSubDomain(MasterDomain.Frc, outlet)
	// // b.mdldir = outdir
	// dt, y, ep, obs, intvl, nstep := b.getForcings()
	// // b.cid0 = -1
	// // if len(b.rtr.SwsCidXR) == 1 {
	// // 	b.rtr.SwsCidXR = map[int][]int{-1: b.cids}
	// // 	b.rtr.Sws = make(map[int]int, b.ncid)
	// // 	for _, c := range b.cids {
	// // 		b.rtr.Sws[c] = -1
	// // 	}
	// // }
	fmt.Printf(" catchment area: %.1f km² (%s cells)\n", b.contarea/1000./1000., mmio.Thousands(int64(b.ncid)))

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())

	tt := mmio.NewTimer()
	fmt.Printf(" number of subwatersheds: %d\n", len(b.rtr.SwsCidXR))
	fmt.Printf(" running %d samples from %d dimensions..\n", nsmpl, nSmplDim)

	printParams := func(m, kstrm, mcasc, soildepth, kfact, dinc float64, mdir string) {
		// tw, err := mmio.NewTXTwriter(mondir + "params.txt")
		tw, err := mmio.NewTXTwriter(mdir + "params.txt")
		defer tw.Close()
		if err != nil {
			log.Fatalf("SampleMaster error: %v", err)
		}
		tw.WriteLine(mmio.MMtime(time.Now()))
		// tw.WriteLine(mondir)
		tw.WriteLine(mdir)
		tw.WriteLine(fmt.Sprintf("m\t%f", m))
		tw.WriteLine(fmt.Sprintf("kstrm\t%f", kstrm))
		tw.WriteLine(fmt.Sprintf("mcasc\t%f", mcasc))
		// tw.WriteLine(fmt.Sprintf("hmax\t%f", hmax))
		// tw.WriteLine(fmt.Sprintf("smax\t%f", smax))
		tw.WriteLine(fmt.Sprintf("dinc\t%f", dinc))
		tw.WriteLine(fmt.Sprintf("soildepth\t%f", soildepth))
		tw.WriteLine(fmt.Sprintf("kfact\t%f", kfact))
	}

	// gen := func(u []float64) {
	// 	setMCdir()
	// 	// m, hmax, smax, dinc, soildepth, kfact := par6(u)
	// 	m, grng, soildepth, kfact := par4(u)
	// 	go printParams(m, grng, soildepth, kfact)
	// 	smpl := b.toDefaultSample(m, grng, soildepth, kfact)
	// 	b.evaluate(&smpl, 0., m, false)
	// 	WaitMonitors()
	// 	compressMC()
	// }
	gen := func(u []float64) float64 {
		mdir := newMCdir()
		// m, hmax, smax, dinc, soildepth, kfact := par6(u)
		// m, grng, soildepth, kfact := Par4(u)
		m, gdn, kstrm, mcasc, soildepth, kfact, dinc := Par7(u)
		go printParams(m, kstrm, mcasc, soildepth, kfact, dinc, mdir)
		smpl := b.toDefaultSample(m, gdn, kstrm, mcasc, soildepth, kfact)
		smpl.dir = mdir
		of := b.evaluate(&smpl, dinc, m, false)
		WaitMonitors()
		compressMC2(mdir)
		fmt.Print(".")
		return of
	}

	sp := smpln.NewLHC(rng, nsmpl, nSmplDim, false)
	for k := 0; k < nsmpl; k++ {
		ut := make([]float64, nSmplDim)
		for j := 0; j < nSmplDim; j++ {
			ut[j] = sp.U[j][k]
		}
		gen(ut)
		fmt.Print(".")
	}

	// montecarlo.GenerateSamples(gen, nSmplDim, nsmpl)
	// tt.Lap("\nsampling complete")

	// u, f, d := montecarlo.RankedUnBiased(gen, nSmplDim, nsmpl)

	// tt.Lap("\nsampling complete")
	// t, err := mmio.NewTXTwriter(mcdir + "MCsummary.csv")
	// if err != nil {
	// 	log.Fatalf("SampleDefault %s save error: %v", mcdir+"MCsummary.csv", err)
	// }
	// t.WriteLine(fmt.Sprintf("rank(of %d),eval,m,smax,dinc,soildepth,kfact", nsmpl))
	// for i, dd := range d {
	// 	m, grng, soildepth, kfact := par4(u[dd])
	// 	t.WriteLine(fmt.Sprintf("%d,%f,%f,%f,%f,%f", i+1, 1.-f[dd], m, grng, soildepth, kfact))
	// }
	// t.Close()
	tt.Lap("results save complete")
}

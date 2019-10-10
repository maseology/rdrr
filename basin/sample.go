package basin

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
	"github.com/maseology/goHydro/met"
	"github.com/maseology/mmio"
	"github.com/maseology/montecarlo"
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

// SampleDefault solves a default-parameter model to a given basin outlet
// changes only 3 basin-wide parameters (Qo, topm, fcasc); freeboard set to 0.
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

	fmt.Printf(" catchment area: %.1f km²\n", b.contarea/1000./1000.)
	fmt.Printf(" building sample HRUs and TOPMODEL\n\n")

	rng := rand.New(mrg63k3a.New())
	rng.Seed(time.Now().UnixNano())
	ver := b.eval

	gen := func(u []float64) float64 {
		m, smax, dinc, soildepth, kfact := par5(u)
		smpl := b.toDefaultSample(m, smax, soildepth, kfact)
		return ver(&smpl, dinc, m, false)
	}

	fmt.Printf(" running %d samples from %d dimensions..\n", nsmpl, nSmplDim)
	u, f, d := montecarlo.RankedUnBiased(gen, nSmplDim, nsmpl)

	v := func() float64 {
		nstep, dtb, dte, intvl := b.frc.trimFrc(-1)
		h2cms := b.contarea / float64(intvl) // [m/ts] to [m³/s] conversion factor
		o, i := make([]float64, nstep), 0
		for d := dtb; !d.After(dte); d = d.Add(time.Second * time.Duration(intvl)) {
			v := b.frc.c[d]
			o[i] = v[met.UnitDischarge] * h2cms
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
	}()

	t, err := mmio.NewTXTwriter(outdir + "sample.csv")
	defer t.Close()
	if err != nil {
		log.Fatalf(" Definition.SaveAs: %v", err)
	}
	t.WriteLine(fmt.Sprintf("rank(of %d),eval,m,smax,dinc,soildepth,kfact", nsmpl))
	for i, dd := range d {
		nse := 1. - math.Pow(f[dd], 2.)/v // converting to nash-sutcliffe
		m, smax, dinc, soildepth, kfact := par5(u[dd])
		t.WriteLine(fmt.Sprintf("%d,%f,%f,%f,%f,%f,%f", i+1, nse, m, smax, dinc, soildepth, kfact))
	}
}

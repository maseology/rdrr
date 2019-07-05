package basin

import (
	"fmt"
	"log"
	"math"
	"sort"
	"sync"

	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
	"github.com/maseology/mmaths"
)

const (
	twoThirds  = 2. / 3.
	fiveThirds = 5. / 3.
)

type sample struct {
	ws     hru.WtrShd        // hru watershed
	gw     map[int]*gwru.TMQ // topmodel
	p0, p1 map[int]float64
}

func topm(u float64) float64 { // TOPMODEL m
	return mmaths.LogLinearTransform(0.001, 10., u)
}

func dsoil(u float64) float64 { // depth of soil [m]
	return mmaths.LinearTransform(0.01, 1., u)
}

func dpsto(u float64) float64 { // impervious depression storage [m]
	return mmaths.LogLinearTransform(0.0001, 0.001, u)
}

func intsto(u float64) float64 { // short and tall vegetation interception storage
	return mmaths.LinearTransform(0.0001, 0.004, u)
}

func mann(u float64) float64 {
	return mmaths.LogLinearTransform(0.0001, 100., u)
}
func fc(u float64) float64 {
	return mmaths.LinearTransform(0.01, 0.4, u)
}

func (b *subdomain) printParam(u ...float64) {
	// transform sample space
	ksg := 2
	fmt.Printf("topm\t\t%.5f\t(%.5f)\n", topm(u[0]), u[0])
	fmt.Printf("dsoil\t\t%.5f\t(%.5f)\n", dsoil(u[1]), u[1])
	// fmt.Printf("dpsto\t\t%.5f\t(%.5f)\n", dpsto(u[2]), u[2])
	// fmt.Printf("intsto\t\t%.5f\t(%.5f)\n", intsto(u[3]), u[3])

	// sample surficial geology types
	nsg, i := 3, 0
	keys := make([]int, 0)
	for k := range b.mpr.sg {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	sdf := b.mpr.sg
	for _, k := range keys {
		sg := sdf[k]
		ksat, por, _ := sg.Sample(u[ksg+nsg*i], u[ksg+nsg*i+1])
		fmt.Printf(" === SG %d\n", k)
		fmt.Printf("ksat\t\t%.2e\t(%.5f)\n", ksat, u[ksg+nsg*i])
		fmt.Printf("por\t\t%.5f\t(%.5f)\n", por, u[ksg+nsg*i+1])
		fmt.Printf("fc\t\t%.5f\t(%.5f)\n", fc(u[ksg+nsg*i+2]), u[ksg+nsg*i+2])
		i++
	}

	// sample landuse types
	klu, nlu, i := ksg+len(b.mpr.sg)*nsg, 1, 0
	keys = make([]int, 0)
	for k := range b.mpr.lu {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		lu := b.mpr.lu[k]
		fmt.Printf(" === LU %d\n", k)
		fmt.Printf("fimp\t\t%.5f\n", lu.Fimp)
		fmt.Printf("intfct\t\t%.5f\n", lu.Intfct)
		fmt.Printf("mann\t\t%.5f\t(%.5f)\n", mann(u[klu+nlu*i]), u[klu+nlu*i])
		i++
	}
}

func (b *subdomain) toSampleU(u ...float64) sample {
	var wg sync.WaitGroup

	ws := make(hru.WtrShd, b.ncid)
	var gw map[int]*gwru.TMQ
	// str := make([]string, 0, len(u))

	// transform sample space
	ksg := 2
	// str = append(str, "topm", "dsoil", "dpsto", "itsto")
	topm := topm(u[0])
	dsoil := dsoil(u[1])
	// dpsto := dpsto( u[2])
	// itsto := intsto(u[3]) // short and tall vegetation interception

	// sample surficial geology types
	nsg, i := 3, 0
	pksat, ppor, pfc := make(map[int]float64, len(b.mpr.sg)), make(map[int]float64, len(b.mpr.sg)), make(map[int]float64, len(b.mpr.sg))
	keys := make([]int, 0)
	for k := range b.mpr.sg {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	sdf := b.mpr.sg
	for _, k := range keys {
		sg := sdf[k]
		pksat[k], ppor[k], _ = sg.Sample(u[ksg+nsg*i], u[ksg+nsg*i+1])
		pfc[k] = fc(u[ksg+nsg*i+2])
		// str = append(str, fmt.Sprintf("%d:ksat", k), fmt.Sprintf("%d:por", k), fmt.Sprintf("%d:fc", k))
		i++
	}

	// sample landuse types
	klu, nlu, i := ksg+len(b.mpr.sg)*nsg, 1, 0
	pn, pfimp, pinfct := make(map[int]float64, len(b.mpr.lu)), make(map[int]float64, len(b.mpr.lu)), make(map[int]float64, len(b.mpr.lu))
	keys = make([]int, 0)
	for k := range b.mpr.lu {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		lu := b.mpr.lu[k]
		pfimp[k] = lu.Fimp
		pinfct[k] = lu.Intfct
		pn[k] = mann(u[klu+nlu*i])
		// str = append(str, fmt.Sprintf("%d:mann", k))
		i++
	}

	n := make(map[int]float64)
	ts := b.frc.h.IntervalSec()
	assignHRUs := func() {
		defer wg.Done()
		ksat := make(map[int]float64)
		var recurs func(int)
		recurs = func(cid int) {
			var hnew hru.HRU
			ksat[cid] = pksat[b.mpr.isg[cid]]
			n[cid] = pn[b.mpr.ilu[cid]]
			drnsto := ppor[b.mpr.isg[cid]] * (1. - pfc[b.mpr.isg[cid]]) * dsoil
			srfsto := ppor[b.mpr.isg[cid]] * pfc[b.mpr.isg[cid]] * dsoil //+ itsto*pinfct[b.mpr.ilu[cid]]*(1.-pfimp[b.mpr.ilu[cid]]) + dpsto*pfimp[b.mpr.ilu[cid]]
			hnew.Initialize(drnsto, srfsto, pfimp[b.mpr.ilu[cid]], ksat[cid], ts)
			ws[cid] = &hnew
			for _, upcid := range b.strc.t.UpIDs(cid) {
				recurs(upcid)
			}
		}
		recurs(b.cid0)
	}
	buildTopmodel := func() {
		defer wg.Done()
		ksat := make(map[int]float64)
		var recurs func(int)
		recurs = func(cid int) {
			ksat[cid] = pksat[b.mpr.isg[cid]] * ts // [m/ts]
			for _, upcid := range b.strc.t.UpIDs(cid) {
				recurs(upcid)
			}
		}
		recurs(b.cid0)

		if b.frc.Q0 <= 0. {
			log.Fatalf("toDefaultSample.buildTopmodel error, initial flow for TOPMODEL (Q0) is set to %v", b.frc.Q0)
		}
		medQ := b.frc.Q0 * b.strc.a * float64(len(ksat)) // [m/d] to [m³/d]
		gw = make(map[int]*gwru.TMQ, 1)
		print("fix")
		gw[0].New(ksat, b.strc.t, b.strc.w, medQ, 2*medQ, topm)
	}

	wg.Add(2)
	go assignHRUs()
	go buildTopmodel()
	wg.Wait()

	// fmt.Println(str)

	return sample{
		ws: ws,
		gw: gw,
		p0: b.buildC0(n, ts),
		p1: b.buildC2(n, ts),
	}
}

func (b *subdomain) buildC0(n map[int]float64, ts float64) map[int]float64 {
	c0 := make(map[int]float64, len(b.cids))
	for _, cid := range b.cids {
		c := fiveThirds * math.Sqrt(b.strc.t.TEC[cid].S) * ts / b.strc.w / n[cid]
		c0[cid] = c / (1. + c)
	}
	return c0
}

func (b *subdomain) buildC2(n map[int]float64, ts float64) map[int]float64 {
	c2 := make(map[int]float64, len(b.cids))
	for _, cid := range b.cids {
		c := fiveThirds * math.Sqrt(b.strc.t.TEC[cid].S) * ts / b.strc.w / n[cid]
		c2[cid] = 1. / (1. + c)
	}
	return c2
}

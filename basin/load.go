package basin

import (
	"fmt"
	"log"
	"math"
	"reflect"
	"sync"

	"github.com/im7mortal/UTM"
	"github.com/maseology/goHydro/grid"
	"github.com/maseology/goHydro/gwru"
	"github.com/maseology/goHydro/hru"
	"github.com/maseology/goHydro/met"
	"github.com/maseology/goHydro/solirrad"
	"github.com/maseology/goHydro/tem"
	"github.com/maseology/mmaths"
	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/lusg"
)

// Loader holds the required input filepaths
type Loader struct {
	metfp, indir, gdfn, temfn, lufn, sgfn string
	outlet                                int
}

// FRC holds forcing data
type FRC struct {
	c met.Coll
	h met.Header
}

// MDL holds structural data
type MDL struct {
	b    hru.Basin
	g    gwru.TMQ
	t    tem.TEM
	f    map[int][366]float64
	a, w float64
}

// NewLoader returns a default Loader
func NewLoader(rootdir string, outlet int) *Loader {
	// lout := Loader{
	// 	metfp:  rootdir + "02EC018.met",
	// 	indir:  rootdir,
	// 	gdfn:   "ORMGP_50_hydrocorrect.uhdem.gdef",
	// 	temfn:  "ORMGP_50_hydrocorrect.uhdem",
	// 	lufn:   "ORMGP_50_hydrocorrect_SOLRISv2_ID.grd",
	// 	sgfn:   "ORMGP_50_hydrocorrect_PorousMedia_ID.grd",
	// 	outlet: -1, // <0: from .met index, 0: no outlet, >0: outlet cell ID
	// }
	lout := Loader{
		metfp:  rootdir + "02EC018.met",
		indir:  rootdir + "coarse/",
		temfn:  "ORMGP_500_hydrocorrect.uhdem",
		gdfn:   "ORMGP_500_hydrocorrect.uhdem.gdef",
		lufn:   "ORMGP_500_hydrocorrect_SOLRISv2_ID.grd",
		sgfn:   "ORMGP_500_hydrocorrect_PorousMedia_ID.grd",
		outlet: outlet, //127669, // 128667, // <0: from .met index, 0: no outlet, >0: outlet cell ID
	}
	lout.check()
	return &lout
}

func (l *Loader) check() {
	v := reflect.ValueOf(*l)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Type() == reflect.TypeOf("") {
			st1 := v.Field(i).String()
			if st1 != l.indir {
				if _, ok := mmio.FileExists(l.indir + st1); !ok {
					if _, ok := mmio.FileExists(st1); !ok {
						log.Panicf("Loader.check() File does not exist:\n  %s", v.Field(i).String())
					}
				}
			} else {
				if ok := mmio.DirExists(st1); !ok {
					log.Panicf("Loader.check() Directory %s does not exist.\n", v.Field(i).String())
				}
			}
		}
	}
}

func (l *Loader) load(m float64) (FRC, MDL) {
	var wgVar, wgStrc sync.WaitGroup

	// import variables, forcings, etc.
	var dc met.Coll
	var hd met.Header
	readmet := func() {
		defer wgVar.Done()
		m, d, err := met.ReadMET(l.metfp)
		if err != nil {
			log.Fatalln(err)
		}
		dc = d
		hd = *m
	}
	wgVar.Add(1)
	go readmet()

	// import structural data
	var t tem.TEM
	var coord map[int]mmaths.Point
	var lu lusg.LandUseColl
	var sg lusg.SurfGeoColl
	gd, err := grid.ReadGDEF(l.indir + l.gdfn)
	if err != nil {
		log.Fatalf("ReadGDEF: %v", err)
	}
	readtopo := func() {
		defer wgStrc.Done()
		if cc, err := t.New(l.indir + l.temfn); err != nil {
			log.Fatalf("TEM.New: %v", err)
		} else {
			coord = cc
		}
	}
	readLU := func() {
		defer wgStrc.Done()
		lu = *lusg.LoadLandUse(l.indir+l.lufn, gd)
	}
	readSG := func() {
		defer wgStrc.Done()
		sg = *lusg.LoadSurfGeo(l.indir+l.sgfn, gd)
	}

	wgStrc.Add(3)
	go readtopo()
	go readLU()
	go readSG()

	// build HRUs, and GW reservoir
	wgStrc.Wait()
	wgVar.Wait()
	wgVar.Add(1)

	// approximating "baseflow when basin is fully saturated" (TOPMODEL) as median discharge
	medQ := func() float64 {
		defer wgVar.Done()
		a, i := make([]float64, len(dc)), 0
		for _, m := range dc {
			v := m[met.UnitDischarge]
			if !math.IsNaN(v) {
				a[i] = v
				i++
			}
		}
		return mmaths.SliceMedian(a)
	}()

	cid0 := int(hd.Locations[0][0].(int32)) // gauge outlet id
	if l.outlet > 0 {
		cid0 = l.outlet
	}
	ts, nc := hd.IntervalSec(), t.UpCnt(cid0)
	b := make(hru.Basin, nc)
	var g gwru.TMQ
	sif := make(map[int][366]float64, nc)
	assignHRUs := func() {
		defer wgStrc.Done()
		var recurs func(int)
		recurs = func(cid int) {
			if _, ok := lu[cid]; !ok {
				log.Fatalf("assignHRUs error, no LandUse assigned to cell ID %d", cid)
			}
			if _, ok := sg[cid]; !ok {
				log.Fatalf("assignHRUs error, no SurfGeo assigned to cell ID %d", cid)
			}
			var h hru.HRU
			h.Initialize(lu[cid].DrnSto, lu[cid].SrfSto, lu[cid].Fimp, sg[cid].Ksat, ts)
			b[cid] = &h
			for _, upcid := range t.UpIDs(cid) {
				recurs(upcid)
			}
		}
		recurs(cid0)
	}
	buildTopmodel := func() {
		defer wgStrc.Done()
		ksat := make(map[int]float64)
		var recurs func(int)
		recurs = func(cid int) {
			if s, ok := sg[cid]; ok {
				ksat[cid] = s.Ksat * ts // [m/d]
				for _, upcid := range t.UpIDs(cid) {
					recurs(upcid)
				}
			} else {
				log.Fatalf("buildTopmodel error, no SurfGeo assigned to cell ID %d", cid)
			}
		}
		recurs(cid0)
		medQ *= gd.CellArea() * float64(len(ksat)) // [m/d] to [m³/d]
		g.New(ksat, t.SubSet(cid0), gd.CellWidth(), medQ, 2*medQ, m)
	}
	buildSolIrradFrac := func() {
		defer wgStrc.Done()
		var utmzone int
		switch hd.ESPG {
		case 26917: // UTM zone 17N
			utmzone = 17
		default:
			log.Fatalf("buildSolIrradFrac error, unknown ESPG code specified %d", hd.ESPG)
		}
		var recurs func(int)
		recurs = func(cid int) {
			if tec, ok := t.TEC[cid]; ok {
				latitude, _, err := UTM.ToLatLon(coord[cid].X, coord[cid].Y, utmzone, "", true)
				if err != nil {
					log.Fatalf("buildSolIrradFrac error, no TEC assigned to cell ID %d", cid)
				}
				si := solirrad.New(latitude, math.Tan(tec.S), math.Pi/2.-tec.A)
				sif[cid] = si.PSIfactor()
				for _, upcid := range t.UpIDs(cid) {
					recurs(upcid)
				}
			} else {
				log.Fatalf("buildSolIrradFrac error, no TEC assigned to cell ID %d", cid)
			}
		}
		recurs(cid0)
	}

	wgVar.Wait()
	wgStrc.Add(3)
	fmt.Println("\nbuilding HRUs, potential solar irradiation and TOPMODEL")
	go assignHRUs()
	go buildTopmodel()
	go buildSolIrradFrac()

	wgStrc.Wait()

	frc := FRC{
		c: dc,
		h: hd,
	}
	mdl := MDL{
		b: b,
		g: g,
		t: t,
		f: sif,
		a: gd.CellArea(),
		w: gd.CellWidth(),
	}

	if l.outlet == -1 {
		l.outlet = cid0
	}
	return frc, mdl
}

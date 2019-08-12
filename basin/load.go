package basin

import (
	"fmt"
	"log"
	"math"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/im7mortal/UTM"
	"github.com/maseology/goHydro/grid"
	"github.com/maseology/goHydro/met"
	"github.com/maseology/goHydro/solirrad"
	"github.com/maseology/goHydro/tem"
	"github.com/maseology/mmio"
	"github.com/maseology/rdrr/lusg"
)

// Loader holds the required input filepaths
type Loader struct{ Dir, Fmet, Fgd, Fhdem, Fsws, Flu, Fsg string }

// LoaderDefault returns a default Loader
func LoaderDefault(rootdir string, outlet int) *Loader {
	// lout := Loader{
	// 	metfp:  rootdir + "02EC018.met",
	// 	indir:  rootdir,
	// 	gdfn:   "ORMGP_50_hydrocorrect.uhdem.gdef",
	// 	temfn:  "ORMGP_50_hydrocorrect.uhdem",
	// 	lufn:   "ORMGP_50_hydrocorrect_SOLRISv2_ID.grd",
	// 	sgfn:   "ORMGP_50_hydrocorrect_PorousMedia_ID.grd",
	// 	outlet: -1, // <0: from .met index, 0: no outlet, >0: outlet cell ID
	// }
	rtcoarse := rootdir + "coarse/"
	lout := Loader{
		Fmet:  rootdir + "02EC018.met",
		Dir:   rtcoarse,
		Fhdem: rtcoarse + "ORMGP_500_hydrocorrect.uhdem",
		Fgd:   rtcoarse + "ORMGP_500_hydrocorrect.uhdem.gdef",
		Flu:   rtcoarse + "ORMGP_500_hydrocorrect_SOLRISv2_ID.grd",
		Fsg:   rtcoarse + "ORMGP_500_hydrocorrect_PorousMedia_ID.grd",
		// Outlet: outlet, //127669, // 128667, // <0: from .met index, 0: no outlet, >0: outlet cell ID
	}
	lout.check()
	return &lout
}

func (l *Loader) check() {
	v := reflect.ValueOf(*l)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Type() == reflect.TypeOf("") {
			st1 := v.Field(i).String()
			if st1 != l.Dir {
				if _, ok := mmio.FileExists(l.Dir + st1); !ok {
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

func (l *Loader) load() (*FORC, STRC, MAPR, *grid.Definition) {
	var wg sync.WaitGroup

	// import forcings
	var frc *FORC
	readmet := func() {
		defer wg.Done()
		if len(l.Fmet) > 0 {
			fmt.Printf(" loading: %s\n", l.Fmet)
		}
		frc, _ = loadForcing(l.Fmet, true)
	}

	// import structural data and mapping arrays
	gd, err := grid.ReadGDEF(l.Fgd)
	if err != nil {
		log.Fatalf(" grid.ReadGDEF: %v", err)
	}
	var t tem.TEM
	var lu lusg.LandUseColl
	var sg lusg.SurfGeoColl
	var ilu, isg, sws map[int]int

	readtopo := func() {
		defer wg.Done()
		fmt.Printf(" loading: %s\n", l.Fhdem)
		if err := t.New(l.Fhdem); err != nil {
			log.Fatalf(" TEM.New: %v", err)
		}
	}
	readLU := func() {
		defer wg.Done()
		fmt.Printf(" loading: %s\n", l.Flu)
		var g grid.Indx
		g.LoadGDef(gd)
		g.NewShort(l.Flu, false)
		ulu := g.UniqueValues()
		lu = *lusg.LoadLandUse(ulu)
		ilu = g.Values()
		// g.ToASC(l.Dir+"lu.asc", false)
	}
	readSG := func() {
		defer wg.Done()
		fmt.Printf(" loading: %s\n", l.Fsg)
		var g grid.Indx
		g.LoadGDef(gd)
		g.NewShort(l.Fsg, false)
		usg := g.UniqueValues()
		sg = *lusg.LoadSurfGeo(usg)
		isg = g.Values()
		// g.ToASC(l.Dir+"sg.asc", false)
	}
	readSWS := func() {
		defer wg.Done()
		fmt.Printf(" loading: %s\n", l.Fsws)
		switch filepath.Ext(l.Fsws) {
		case ".imap":
			sws, err = mmio.ReadBinaryIMAP(l.Fsws)
			if err != nil {
				log.Fatalf(" Loader.readSWS error with ReadBinaryIMAP: %v\n\n", err)
			}
			// var g grid.Indx
			// g.LoadGDef(gd)
			// g.NewIMAP(sws)
			// g.ToASC(l.Dir+"sws.asc", false)
		case ".indx":
			var g grid.Indx
			g.LoadGDef(gd)
			g.New(l.Fsws, false)
			sws = g.Values()
			// g.ToASC(l.Dir+"sws.asc", false)
		default:
			log.Fatalf(" Loader.readSWS: unrecognised file type: %s\n", l.Fsws)
		}
	}

	wg.Add(4)
	go readmet()
	go readtopo()
	go readLU()
	go readSG()
	if len(l.Fsws) > 0 {
		wg.Add(1)
		go readSWS()
	}
	wg.Wait()

	// compute static variables
	cid0, nc := -1, gd.Nactives()
	if frc != nil {
		if frc.h.Nloc() == 1 && frc.h.LocationCode() > 0 {
			cid0 = int(frc.h.Locations[0][0].(int32)) // gauge outlet id found in met file
		} else {
			log.Fatalf(" Loader.load error: unrecognized .met type\n")
		}
		if cid0 >= 0 {
			nc = t.UpCnt(cid0) // recount number of cells
		}
	}

	sif := make(map[int][366]float64, nc)
	buildSolIrradFrac := func() {
		defer wg.Done()
		fmt.Printf(" building potential solar irradiation field\n")
		var utmzone int
		if frc != nil {
			switch frc.h.ESPG {
			case 26917: // UTM zone 17N
				utmzone = 17
			default:
				log.Fatalf(" buildSolIrradFrac error, unknown ESPG code specified %d", frc.h.ESPG)
			}
		} else {
			utmzone = 17 // UTM zone 17N (by default)
		}

		type kv struct {
			k int
			v [366]float64
		}
		var wg1 sync.WaitGroup
		ch := make(chan kv, nc)
		psi := func(tec tem.TEC, cid int) {
			defer wg1.Done()
			latitude, _, err := UTM.ToLatLon(gd.Coord[cid].X, gd.Coord[cid].Y, utmzone, "", true)
			if err != nil {
				log.Fatalf(" buildSolIrradFrac error: %v -- (x,y)=(%f, %f); cid: %d\n", err, gd.Coord[cid].X, gd.Coord[cid].Y, cid)
			}
			si := solirrad.New(latitude, math.Tan(tec.S), math.Pi/2.-tec.A)
			ch <- kv{k: cid, v: si.PSIfactor()}
		}

		if cid0 >= 0 {
			var recurs func(int)
			recurs = func(cid int) {
				if tec, ok := t.TEC[cid]; ok {
					wg1.Add(1)
					go psi(tec, cid)
					for _, upcid := range t.UpIDs(cid) {
						recurs(upcid)
					}
				} else {
					log.Fatalf(" buildSolIrradFrac (recurse) error, no TEC assigned to cell ID %d", cid)
				}
			}
			recurs(cid0)
		} else {
			for _, cid := range gd.Actives() {
				if tec, ok := t.TEC[cid]; ok {
					wg1.Add(1)
					go psi(tec, cid)
				} else {
					log.Fatalf(" buildSolIrradFrac error, no TEC assigned to cell ID %d", cid)
				}
			}
		}
		wg1.Wait()
		close(ch)
		for kv := range ch {
			sif[kv.k] = kv.v
		}
	}

	wg.Add(1)
	go buildSolIrradFrac()
	wg.Wait()

	mdl := STRC{
		t: t,
		f: sif,
		a: gd.CellArea(),
		w: gd.CellWidth(),
	}
	mpr := MAPR{
		lu:  lu,
		sg:  sg,
		sws: sws,
		ilu: ilu,
		isg: isg,
	}

	return frc, mdl, mpr, gd
}

// LoadForcing (re-)loads forcing data
func loadForcing(fp string, print bool) (*FORC, int) {
	// import forcings
	if _, ok := mmio.FileExists(fp); !ok {
		return nil, -1
	}
	m, d, err := met.ReadMET(fp, print)
	if err != nil {
		log.Fatalln(err)
	}

	// checks
	dtb, dte, intvl := m.BeginEndInterval() // start date, end date, time step interval [s]
	for dt := dtb; !dt.After(dte); dt = dt.Add(time.Second * time.Duration(intvl)) {
		v := d[dt]
		// y := v[met.AtmosphericYield]     // precipitation/atmospheric yield (rainfall + snowmelt)
		ep := v[met.AtmosphericDemand] // evaporative demand
		if ep < 0. {
			d[dt][met.AtmosphericDemand] = 0.
		}
	}

	if m.Nloc() != 1 && m.LocationCode() <= 0 {
		log.Fatalf(" basin.loadForcing error: unrecognized .met type\n")
	}
	outlet := int(m.Locations[0][0].(int32))

	return &FORC{
		c: d,  // met.Coll
		h: *m, // met.Header
	}, outlet
}

// masterForcing returns forcing data from mastreDomain
func masterForcing() (*FORC, int) {
	if masterDomain.frc.h.Nloc() != 1 && masterDomain.frc.h.LocationCode() <= 0 {
		log.Fatalf(" basin.masterForcing error: invalid *FROC type in masterDomain\n")
	}
	return masterDomain.frc, int(masterDomain.frc.h.Locations[0][0].(int32))
}

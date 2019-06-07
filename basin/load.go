package basin

import (
	"fmt"
	"log"
	"math"
	"reflect"
	"sync"

	"github.com/im7mortal/UTM"
	"github.com/maseology/goHydro/grid"
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

func (l *Loader) load() (FORC, STRC, MAPR, *grid.Definition) {
	var wg sync.WaitGroup

	// import forcings
	var dc met.Coll
	var hd met.Header
	readmet := func() {
		defer wg.Done()
		m, d, err := met.ReadMET(l.metfp)
		if err != nil {
			log.Fatalln(err)
		}
		dc = d
		hd = *m
	}

	// import structural data and mapping arrays
	gd, err := grid.ReadGDEF(l.indir + l.gdfn)
	if err != nil {
		log.Fatalf("ReadGDEF: %v", err)
	}
	var t tem.TEM
	var coord map[int]mmaths.Point
	var lu lusg.LandUseColl
	var sg lusg.SurfGeoColl
	var ilu, isg map[int]int
	readtopo := func() {
		defer wg.Done()
		if cc, err := t.New(l.indir + l.temfn); err != nil {
			log.Fatalf("TEM.New: %v", err)
		} else {
			coord = cc
		}
	}
	readLU := func() {
		defer wg.Done()
		fmt.Printf(" loading: %s\n", l.indir+l.lufn)
		var g grid.Indx
		g.LoadGDef(gd)
		g.NewShort(l.indir+l.lufn, false)
		ulu := g.UniqueValues()
		lu = *lusg.LoadLandUse(ulu)
		ilu = g.Values()
	}
	readSG := func() {
		defer wg.Done()
		fmt.Printf(" loading: %s\n", l.indir+l.sgfn)
		var g grid.Indx
		g.LoadGDef(gd)
		g.NewShort(l.indir+l.sgfn, false)
		usg := g.UniqueValues()
		sg = *lusg.LoadSurfGeo(usg)
		isg = g.Values()
	}

	wg.Add(4)
	go readmet()
	go readtopo()
	go readLU()
	go readSG()
	wg.Wait()

	// compute static variables
	cid0 := int(hd.Locations[0][0].(int32)) // gauge outlet id
	if l.outlet > 0 {
		cid0 = l.outlet
	}
	nc := t.UpCnt(cid0)
	sif := make(map[int][366]float64, nc)
	buildSolIrradFrac := func() {
		defer wg.Done()
		fmt.Printf("\n building potential solar irradiation\n")
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

	wg.Add(1)
	go buildSolIrradFrac()
	wg.Wait()

	frc := FORC{
		c: dc,
		h: hd,
	}
	mdl := STRC{
		t: t,
		f: sif,
		a: gd.CellArea(),
		w: gd.CellWidth(),
	}
	mpr := MAPR{
		lu:  lu,
		sg:  sg,
		ilu: ilu,
		isg: isg,
	}

	if l.outlet == -1 {
		l.outlet = cid0
	}
	return frc, mdl, mpr, gd
}

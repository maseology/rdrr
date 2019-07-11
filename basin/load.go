package basin

import (
	"fmt"
	"log"
	"math"
	"path/filepath"
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
	Dir, Fmet, Fgd, Fhdem, Fsws, Flu, Fsg string
	Outlet                                int
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
	rtcoarse := rootdir + "coarse/"
	lout := Loader{
		Fmet:   rootdir + "02EC018.met",
		Dir:    rtcoarse,
		Fhdem:  rtcoarse + "ORMGP_500_hydrocorrect.uhdem",
		Fgd:    rtcoarse + "ORMGP_500_hydrocorrect.uhdem.gdef",
		Flu:    rtcoarse + "ORMGP_500_hydrocorrect_SOLRISv2_ID.grd",
		Fsg:    rtcoarse + "ORMGP_500_hydrocorrect_PorousMedia_ID.grd",
		Outlet: outlet, //127669, // 128667, // <0: from .met index, 0: no outlet, >0: outlet cell ID
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

func (l *Loader) load() (FORC, STRC, MAPR, *grid.Definition) {
	var wg sync.WaitGroup

	// import forcings
	var dc met.Coll
	var hd met.Header
	readmet := func() {
		defer wg.Done()
		m, d, err := met.ReadMET(l.Fmet)
		if err != nil {
			log.Fatalln(err)
		}
		dc = d
		hd = *m
	}

	// import structural data and mapping arrays
	gd, err := grid.ReadGDEF(l.Fgd)
	if err != nil {
		log.Fatalf("ReadGDEF: %v", err)
	}
	var t tem.TEM
	var coord map[int]mmaths.Point
	var lu lusg.LandUseColl
	var sg lusg.SurfGeoColl
	var ilu, isg, sws map[int]int
	readtopo := func() {
		defer wg.Done()
		if cc, err := t.New(l.Fhdem); err != nil {
			log.Fatalf("TEM.New: %v", err)
		} else {
			coord = cc
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
		// g.ToASC("N:/CreditSWAT/lu.asc", false)
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
		// g.ToASC("N:/CreditSWAT/sg.asc", false)
	}
	readSWS := func() {
		defer wg.Done()
		fmt.Printf(" loading: %s\n", l.Fsws)
		print(filepath.Ext(l.Fsws))
		switch filepath.Ext(l.Fsws) {
		case ".imap":
			sws, err = mmio.ReadBinaryIMAP(l.Fsws)
			if err != nil {
				log.Fatalf("Loader.readSWS error with ReadBinaryIMAP\n")
			}
			// var g grid.Indx
			// g.LoadGDef(gd)
			// g.NewIMAP(sws)
			// g.ToASC("N:/CreditSWAT/sws.asc", false)
		case ".indx":
			var g grid.Indx
			g.LoadGDef(gd)
			g.New(l.Flu, true)
			sws = g.Values()
		default:
			log.Fatalf("Loader.readSWS: unrecognised file type: %s\n", l.Fsws)
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
	cid0 := -1
	if len(hd.Locations) > 0 {
		cid0 = int(hd.Locations[0][0].(int32)) // gauge outlet id
	}
	if l.Outlet > 0 {
		cid0 = l.Outlet
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

		// cmpt := func(tec tem.TEC, cid int) {
		// 	latitude, _, err := UTM.ToLatLon(coord[cid].X, coord[cid].Y, utmzone, "", true)
		// 	if err != nil {
		// 		log.Fatalf("buildSolIrradFrac error, no TEC assigned to cell ID %d", cid)
		// 	}
		// 	si := solirrad.New(latitude, math.Tan(tec.S), math.Pi/2.-tec.A)
		// 	sif[cid] = si.PSIfactor()
		// }
		// for k, v := range t.TEC {
		// 	cmpt(v, k)
		// }

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
		sws: sws,
		ilu: ilu,
		isg: isg,
	}

	if l.Outlet == -1 {
		l.Outlet = cid0
	}
	return frc, mdl, mpr, gd
}

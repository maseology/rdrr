package basin

import (
	"encoding/gob"
	"log"
	"math"
	"os"
	"sync"

	"github.com/im7mortal/UTM"
	"github.com/maseology/goHydro/grid"
	"github.com/maseology/goHydro/solirrad"
	"github.com/maseology/goHydro/tem"
)

// loadSolIrradFrac builds slope-aspect corrections for every cell
func loadSolIrradFrac(frc *FORC, t *tem.TEM, gd *grid.Definition, nc, cid0 int, EnableSineET bool) map[int][366]float64 {
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
		if EnableSineET {
			// returns Sine-curve potential evaporation
			ep := si.PSIfactor()
			for j := 0; j < 366; j++ {
				ep[j] *= sinEp(j)
			}
			ch <- kv{k: cid, v: ep}
		} else {
			ch <- kv{k: cid, v: si.PSIfactor()}
		}
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
	f := make(map[int][366]float64, nc)
	for kv := range ch {
		f[kv.k] = kv.v
	}
	return f
}

// sifSave sif to gob
func sifSave(fp string, sif map[int][366]float64) error {
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		return err
	}
	enc := gob.NewEncoder(f)
	err = enc.Encode(sif)
	if err != nil {
		return err
	}
	return nil
	// buf := new(bytes.Buffer)
	// for k, v := range sif {
	// 	if err := binary.Write(buf, binary.LittleEndian, int32(k)); err != nil {
	// 		log.Fatalln("sifSave failed:", err)
	// 	}
	// 	for i := 0; i < 366; i++ {
	// 		if err := binary.Write(buf, binary.LittleEndian, v[i]); err != nil {
	// 			log.Fatalln("sifSave failed:", err)
	// 		}
	// 	}
	// }
	// if err := ioutil.WriteFile(fp, buf.Bytes(), 0644); err != nil { // see: https://en.wikipedia.org/wiki/File_system_permissions
	// 	return fmt.Errorf(" sifSave failed: %v", err)
	// }
	// return nil
}

// sifLoad sif gob
func sifLoad(fp string) (map[int][366]float64, error) {
	var sif map[int][366]float64
	f, err := os.Open(fp)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	enc := gob.NewDecoder(f)
	err = enc.Decode(&sif)
	if err != nil {
		return nil, err
	}
	return sif, nil
	// var err error
	// b, err := ioutil.ReadFile(fp)
	// if err != nil {
	// 	return nil, fmt.Errorf("sifLoad: ioutil.ReadFile failed: %v", err)
	// }
	// buf := bytes.NewReader(b)
	// n := len(b) / (4 + 366*8)
	// m := make(map[int][366]float64, n)
	// type v struct {
	// 	i int32
	// 	a [366]float64
	// }
	// vs := make([]v, 2*n)
	// if err := binary.Read(buf, binary.LittleEndian, vs); err != nil {
	// 	return nil, fmt.Errorf("sifLoad: binary.Read failed: %v", err)
	// }
	// for _, v := range vs {
	// 	m[int(v.i)] = v.a
	// }
	// return m, nil
}

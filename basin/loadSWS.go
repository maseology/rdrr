package basin

import (
	"log"
	"path/filepath"

	"github.com/maseology/goHydro/grid"
	"github.com/maseology/mmio"
)

// loadSWS loads subwatershed info
func loadSWS(gd *grid.Definition, fp string) (sws, dsws map[int]int, swscidxr map[int][]int) {
	switch filepath.Ext(fp) {
	case ".imap":
		var err error
		sws, err = mmio.ReadBinaryIMAP(fp)
		if err != nil {
			log.Fatalf(" Loader.readSWS.loadSWS error with ReadBinaryIMAP: %v\n\n", err)
		}
	case ".indx":
		var g grid.Indx
		g.LoadGDef(gd)
		g.New(fp, false)
		sws = g.Values()
	default:
		log.Fatalf(" Loader.readSWS: unrecognized file type: %s\n", fp)
	}
	// collect sws ids
	sct := make(map[int][]int, len(sws))
	for c, s := range sws {
		if _, ok := sct[s]; ok {
			sct[s] = append(sct[s], c)
		} else {
			sct[s] = []int{c}
		}
	}
	swscidxr = make(map[int][]int, len(sct))
	for k, v := range sct {
		a := make([]int, len(v))
		copy(a, v)
		swscidxr[k] = a
	}
	// collect topology
	if _, ok := mmio.FileExists(mmio.RemoveExtension(fp) + ".topo"); ok {
		d, err := mmio.ReadCSV(mmio.RemoveExtension(fp) + ".topo")
		if err != nil {
			log.Fatalf(" Loader.readSWS: error reading %s: %v\n", mmio.RemoveExtension(fp)+".topo", err)
		}
		dsws = make(map[int]int, len(d)) // note: swsids not contained within dsws drain to farfield
		for _, ln := range d {
			dsws[int(ln[1])] = int(ln[2]) // linkID,upstream_swsID,downstream_swsID
		}
	}
	return
}

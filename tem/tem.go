package tem

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"path/filepath"

	"github.com/maseology/mmio"
)

// TEM topologic elevation model
type TEM struct {
	TECs map[int]*TEC
	us   map[int][]int
	c    int
}

// NumCells number of cells that make up the TEM
func (t *TEM) NumCells() int {
	return len(t.TECs)
}

// New contructor
func (t *TEM) New(fp string) {
	fmt.Printf(" loading: %s\n", fp)

	switch filepath.Ext(fp) {
	case ".uhdem", ".bin":
		t.loadUHDEM(fp)
	default:
		panic(" error: unknown TEM file type used")
	}

	t.buildUpslopes()
}

func (t *TEM) loadUHDEM(fp string) {
	// load file
	buf := mmio.OpenBinary(fp)

	// check file type
	switch mmio.ReadString(buf) {
	case "unstructured":
		// do nothing
	default:
		log.Fatalln("Fatal error: unsupported UHDEM type")
	}

	// read dem data
	var nc int32
	binary.Read(buf, binary.LittleEndian, &nc) // number of cells
	t.TECs = make(map[int]*TEC)
	for i := int32(0); i < nc; i++ {
		u := uhdemReader{}
		u.uhdemRead(buf)
		t.TECs[int(u.I)] = u.toTEC()
	}

	// read flowpaths
	var nfp int32
	binary.Read(buf, binary.LittleEndian, &nfp) // number of flowpaths
	for i := int32(0); i < nfp; i++ {
		f := fpReader{}
		f.fpRead(buf)
		t.TECs[int(f.I)].ds = int(f.Ids)
	}
}

func (t *TEM) buildUpslopes() {
	t.us = make(map[int][]int)
	for i, v := range t.TECs {
		if v.ds >= 0 {
			t.us[v.ds] = append(t.us[v.ds], i)
		}
	}
}

// UnitContributingArea computes the (unit) contributing area from a given cell id
func (t *TEM) UnitContributingArea(cid int) float64 {
	t.c = 0
	t.climb(cid)
	return float64(t.c)
}

func (t *TEM) climb(cid int) {
	t.c++
	for _, i := range t.us[cid] {
		t.climb(i)
	}
}

// TEC topologic elevation model cell
type TEC struct {
	Z, S, A float64
	ds      int
}

// New constructor
func (t *TEC) New(z, s, a float64, ds int) {
	t.Z = z   // elevation
	t.S = s   // slope
	t.A = a   // aspect
	t.ds = ds // downslope id
}

type uhdemReader struct {
	I             int32
	X, Y, Z, S, A float64
}

func (u *uhdemReader) uhdemRead(b *bytes.Reader) {
	err := binary.Read(b, binary.LittleEndian, u)
	if err != nil {
		log.Fatalln("Fatal error: uhdemRead failed", err)
	}
}

func (u *uhdemReader) toTEC() *TEC {
	var t TEC
	t.New(u.Z, u.S, u.A, -1)
	return &t
}

type fpReader struct {
	I, Nds, Ids int32
	F           float64
}

func (f *fpReader) fpRead(b *bytes.Reader) {
	err := binary.Read(b, binary.LittleEndian, f)
	if err != nil {
		log.Fatalln("Fatal error: fpRead failed: ", err)
	}
	if f.Nds != 1 {
		log.Fatalln("Fatal error: fpRead only support singular downslope IDs (i.e., tree-graph topology only).", err)
	}
}

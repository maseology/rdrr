package tem

import (
	"encoding/binary"
	"fmt"
	"log"
	"path/filepath"

	"github.com/maseology/mmio"
)

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

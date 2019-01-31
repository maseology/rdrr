package tem

import (
	"bytes"
	"encoding/binary"
	"log"
)

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

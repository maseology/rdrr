package model

import (
	"fmt"

	"github.com/maseology/mmio"
)

type monitor struct {
	v []float64
	c int
}

func (m *monitor) print(mdir string) {
	defer gwg.Done()
	mmio.WriteFloats(fmt.Sprintf("%s%d.cms", mdir, m.c), m.v) // monitor file (discharge from a cell [m³/s])
	// vv := make([]float64, len(m.v))
	// for k, v := range m.v {
	// 	vv[k] = v * h2cms
	// }
	// mmio.WriteFloats(fmt.Sprintf("%s%d.mon", mondir, m.c), vv)
}

package basin

import (
	"encoding/gob"
	"fmt"
	"os"

	"github.com/maseology/goHydro/tem"
)

// STRC holds model structural data
type STRC struct {
	TEM          *tem.TEM    // topology
	UpCnt        map[int]int // cell upslope count (unit contributing area)
	Acell, Wcell float64     // cell area, cell width
}

// SaveGob STRC to gob
func (s *STRC) SaveGob(fp string) error {
	f, err := os.Create(fp)
	defer f.Close()
	if err != nil {
		return fmt.Errorf(" STRC.Save %v", err)
	}
	if err := gob.NewEncoder(f).Encode(s); err != nil {
		return fmt.Errorf(" STRC.Save %v", err)
	}
	return nil
}

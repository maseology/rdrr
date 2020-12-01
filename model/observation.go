package model

import (
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/maseology/mmio"
)

func (b *subdomain) getObs(obsFP string) error {
	if _, ok := mmio.FileExists(obsFP); !ok {
		return fmt.Errorf("subdomain.getObs() error: file %s does not exist", obsFP)
	}
	f, err := os.Open(obsFP)
	if err != nil {
		return fmt.Errorf("subdomain.getObs() failed: %v", err)
	}
	defer f.Close()

	if math.Mod(86400., b.frc.IntervalSec) != 0. {
		return fmt.Errorf("subdomain.getObs() failed: forcing interval frequency = %f timesteps per day; needs to be an even divisor", 86400./b.frc.IntervalSec)
	}
	nsstp := int(86400. / b.frc.IntervalSec)
	dateToDay := func(dt time.Time) time.Time {
		return time.Date(dt.Year(), dt.Month(), dt.Day(), 0, 0, 0, 0, dt.Location())
	}
	dtb, dte := dateToDay(b.frc.T[0]), dateToDay(b.frc.T[len(b.frc.T)-1])

	vs, ii := make([]float64, len(b.frc.T)), 0
	acell := b.contarea / b.fncid
	cms2h := b.frc.IntervalSec / acell
	for rec := range mmio.LoadCSV(io.Reader(f)) {
		var dt time.Time
		var v float64
		if dt, err = time.Parse("2006-01-02", rec[0]); err != nil {
			return fmt.Errorf("subdomain.getObs() failed: %v", err)
		}
		if dt.Before(dtb) || dt.After(dte) {
			continue
		}
		if v, err = strconv.ParseFloat(rec[1], 64); err != nil {
			return fmt.Errorf("subdomain.getObs() failed: %v", err)
		}
		for i := 0; i < nsstp; i++ {
			vs[ii] = v * cms2h
			ii++
		}
	}

	b.gwsink = 0. // gwsink(mmio.FileName(obsFP, false))
	b.obs = vs
	mmio.WriteFloats(fmt.Sprintf("%s.obs", obsFP), vs)
	return nil
}

// func gwsink(sta string) float64 {
// 	d := map[string]float64{
// 		"02EC021": .0005,
// 		"02ED030": .00025,
// 		"02HB020": .0005,
// 		"02HC056": .0005,
// 		"02HC005": .00025, // m/ts
// 		// "02HJ005": .08,    // m³/s
// 	}
// 	if v, ok := d[sta]; ok {
// 		return v
// 	}
// 	return 0.
// }

package basin

import (
	"log"
	"time"

	"github.com/maseology/goHydro/met"
	"github.com/maseology/mmio"
)

// LoadForcing (re-)loads forcing data
func loadForcing(fp string, print bool) (*FORC, int) {
	// import forcings
	if _, ok := mmio.FileExists(fp); !ok {
		return nil, -1
	}
	m, d, err := met.ReadMET(fp, print)
	if err != nil {
		log.Fatalln(err)
	}

	// checks
	dtb, dte, intvl := m.BeginEndInterval() // start date, end date, time step interval [s]
	temp, k := make([]temporal, m.Nstep()), 0
	x := m.WBDCxr()
	for dt := dtb; !dt.After(dte); dt = dt.Add(time.Second * time.Duration(intvl)) {
		if d.T[k] != dt {
			log.Fatalf("loadForcing error: date mis-match: %v vs %v", d.T[k], dt)
		}
		v := d.D[k][0] // [date ID][cell ID][type ID]
		// y := v[x["AtmosphericYield"]]     // precipitation/atmospheric yield (rainfall + snowmelt)
		ep := v[x["AtmosphericDemand"]] // evaporative demand
		if ep < 0. {
			d.D[k][0][x["AtmosphericDemand"]] = 0.
		}
		temp[k] = temporal{doy: dt.YearDay() - 1, mt: int(dt.Month())}
		k++
	}

	if m.Nloc() != 1 && m.LocationCode() <= 0 {
		log.Fatalf(" basin.loadForcing error: unrecognized .met type\n")
	}
	outlet := int(m.Locations[0][0].(int32))

	return &FORC{
		c:   *d, // met.Coll
		h:   *m, // met.Header
		t:   temp,
		nam: mmio.FileName(fp, false), // station name
	}, outlet
}

// masterForcing returns forcing data from mastreDomain
func masterForcing() (*FORC, int) {
	if masterDomain.frc == nil {
		log.Fatalf(" basin.masterForcing error: masterDomain.frc == nil\n")
	}
	if masterDomain.frc.h.Nloc() != 1 && masterDomain.frc.h.LocationCode() <= 0 {
		log.Fatalf(" basin.masterForcing error: invalid *FORC type in masterDomain\n")
	}
	return masterDomain.frc, int(masterDomain.frc.h.Locations[0][0].(int32))
}

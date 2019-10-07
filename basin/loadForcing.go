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
	for dt := dtb; !dt.After(dte); dt = dt.Add(time.Second * time.Duration(intvl)) {
		v := d[dt]
		// y := v[met.AtmosphericYield]     // precipitation/atmospheric yield (rainfall + snowmelt)
		ep := v[met.AtmosphericDemand] // evaporative demand
		if ep < 0. {
			d[dt][met.AtmosphericDemand] = 0.
		}
	}

	if m.Nloc() != 1 && m.LocationCode() <= 0 {
		log.Fatalf(" basin.loadForcing error: unrecognized .met type\n")
	}
	outlet := int(m.Locations[0][0].(int32))

	return &FORC{
		c:   d,                        // met.Coll
		h:   *m,                       // met.Header
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

package model

import (
	"fmt"

	"github.com/maseology/mmio"
)

// RunDefault runs simulation with fewest parameters
func (d *Domain) RunDefault(mdldir, chkdir string, topm, kstrm, mcasc, soildepth, kfact float64, outlet int, print bool) float64 {
	tt := mmio.NewTimer()

	// build submodel
	b := d.newSubDomain(d.Frc, outlet)
	if print {
		tt.Lap("sub-domain load complete")
		fmt.Printf(" catchment area: %.1f km² (%s cells)\n", b.contarea/1000./1000., mmio.Thousands(int64(b.ncid)))
		fmt.Printf(" building sample HRUs and TOPMODEL\n")
		b.print()
	}

	// add parameterization
	smpl := b.defaultSample(topm, kstrm, mcasc, soildepth, kfact)
	smpl.dir = mdldir
	if print {
		tt.Lap("sample build complete")
		if len(chkdir) > 0 {
			mmio.MakeDir(chkdir)
			b.write(chkdir)
			smpl.write(chkdir)
			tt.Lap("sample maps printed")
		}
		mmio.FileRename("hyd.png", "hydx.png", true)
		fmt.Printf(" number of subwatersheds: %d\n", len(smpl.gw))
		fmt.Printf("\n running model..\n\n")
		// return -1.
	}

	// dt, y, ep, obs, intvl, nstep := b.getForcings()
	of := b.evaluate(&smpl, topm, print, evalWB)
	WaitMonitors()
	return of
}

package model

import (
	"fmt"

	"github.com/maseology/mmio"
)

// RunDefault runs simulation with default parameters
// topm: TOPMODEL m; hmax: max depth of mobile store; slpmax: cell slope above which all cascades; dinc: relative depth of channel incision
func RunDefault(mdldir, chkdir string, topm, hmax, slpmx, dinc, soildepth, kfact float64, outlet int, print bool) float64 {
	tt := mmio.NewTimer()

	// build submodel
	b := masterDomain.newSubDomain(masterDomain.frc, outlet)
	if print {
		tt.Lap("sub-domain load complete")
		fmt.Printf(" catchment area: %.1f km² (%s cells)\n", b.contarea/1000./1000., mmio.Thousands(int64(b.ncid)))
		fmt.Printf(" building sample HRUs and TOPMODEL\n")
	}

	// add parameterization
	smpl := b.toDefaultSample(topm, slpmx, soildepth, kfact)
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
	of := b.evaluate(&smpl, dinc, hmax, topm, print)
	WaitMonitors()
	return of
}

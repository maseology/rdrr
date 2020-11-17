package basin

import (
	"fmt"

	"github.com/maseology/mmio"
)

// RunDefault runs simulation with default parameters
func RunDefault(mdldir, chkdir string, topm, smax, dinc, soildepth, kfact float64, print bool) float64 {
	tt := mmio.NewTimer()
	b := masterDomain.newSubDomain(masterDomain.frc, -1)
	b.mdldir = mdldir

	if print {
		tt.Lap("sub-domain load complete")
		fmt.Printf(" catchment area: %.1f km² (%s cells)\n", b.contarea/1000./1000., mmio.Thousands(int64(b.ncid)))
		fmt.Printf(" building sample HRUs and TOPMODEL\n")
	}
	smpl := b.toDefaultSample(topm, smax, soildepth, kfact)

	if print {
		tt.Lap("sample build complete")
		if len(chkdir) > 0 {
			mmio.MakeDir(chkdir)
			b.write(chkdir)
			smpl.print(chkdir)
			tt.Lap("sample maps printed")
		}
		mmio.FileRename("hyd.png", "hydx.png", true)
		fmt.Printf(" number of subwatersheds: %d\n", len(smpl.gw))
		fmt.Printf("\n running model..\n\n")
	}

	dt, y, ep, obs, intvl, nstep := b.getForcings()
	return b.eval(&smpl, dt, y, ep, obs, intvl, nstep, dinc, topm, print)
}

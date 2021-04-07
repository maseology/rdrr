package model

import (
	"fmt"

	"github.com/maseology/mmio"
)

// RunSurfGeo runs like RunDefault, but adds sampling of surficial geology types to an outlet cellID (=-1 for full-domain model)
func (d *Domain) RunSurfGeo(mdldir, chkdir string, topm, grdMin, kstrm, mcasc, soildepth, dinc, urbDiv float64, ksat []float64, outlet int, print bool) float64 {
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
	smpl := b.surfgeoSample(topm, grdMin, kstrm, mcasc, urbDiv, soildepth, ksat)
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
	of := b.evaluate(&smpl, dinc, topm, print)
	WaitMonitors()
	return of
}

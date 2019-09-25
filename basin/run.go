package basin

import (
	"fmt"
	"log"

	"github.com/maseology/mmio"
)

// RunDefault runs simulation with default parameters
func RunDefault(mdldir, metfp, chkdir string, topm, fcasc, Qo, soildepth float64, print bool) float64 {
	tt := mmio.NewTimer()
	if masterDomain.IsEmpty() {
		log.Fatalf(" basin.RunDefault error: masterDomain is empty\n")
	}
	var b subdomain
	if len(metfp) == 0 {
		if masterDomain.frc == nil {
			log.Fatalf(" basin.RunDefault error: no forcings made available\n")
		}
		b = masterDomain.newSubDomain(masterForcing()) // gauge outlet cell id found in .met file
	} else {
		b = masterDomain.newSubDomain(loadForcing(metfp, print)) // gauge outlet cell id found in .met file
	}
	b.mdldir = mdldir

	if print {
		tt.Lap("sub-domain load complete")
		fmt.Printf(" catchment area: %.1f km² (%s cells)\n", b.contarea/1000./1000., mmio.Thousands(int64(b.ncid)))
		fmt.Printf(" building sample HRUs and TOPMODEL\n")
	}
	smpl := b.toDefaultSample(topm, fcasc, soildepth)

	if print {
		tt.Lap("sample build complete")
		if len(chkdir) > 0 {
			mmio.MakeDir(chkdir)
			masterDomain.gd.SaveAs(chkdir + "masterDomain.gdef")
			b.print(chkdir)
			smpl.print(chkdir)
			tt.Lap("sample map printing")
		}
		mmio.FileRename("hyd.png", "hydx.png", true)
		fmt.Printf(" number of subwatersheds: %d\n", len(smpl.gw))
		fmt.Printf("\n running model..\n\n")
	}

	return b.eval(&smpl, Qo, topm, print)
}

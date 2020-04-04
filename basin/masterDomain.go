package basin

import (
	"fmt"
	"log"
)

// MasterDomain holds all data from which sub-domain scale models can be derived
var masterDomain domain

// LoadMasterDomain loads all data from which sub-domain scale models can be derived
func LoadMasterDomain(ldr *Loader, buildEP bool) {
	fmt.Println("Loading Master Domain..")
	masterDomain = newDomain(ldr, buildEP)
}

// LoadUniformMasterDomain loads all data from which sub-domain scale models can be derived
func LoadUniformMasterDomain(ldr *Loader, buildEP bool) {
	fmt.Println("Loading Master Domain..")
	masterDomain = newUniformDomain(ldr, buildEP)
}

func masterToSubomain(metfp string) (b subdomain, proceed bool) {
	proceed = false
	if masterDomain.IsEmpty() {
		log.Fatalf(" basin.RunDefault error: masterDomain is empty")
	}
	if len(metfp) == 0 {
		if masterDomain.frc == nil {
			log.Fatalf(" basin.RunDefault error: no forcings made available\n")
		}
		b = masterDomain.newSubDomain(masterForcing()) // gauge outlet cell id found in .met file
	} else {
		b = masterDomain.newSubDomain(loadForcing(metfp, true)) // gauge outlet cell id found in .met file
		// if masterDomain.frc != nil && masterDomain.frc.nam == "gob" {
		// 	b = masterDomain.newSubDomain(masterForcingNewOutlet(metfp)) // gauge outlet cell id found in .met file
		// 	if !b.frc.hasObservations() {
		// 		fmt.Println("   >>>>>> model will not proceed as no observations were found within model period")
		// 		return
		// 	}
		// 	dtb, dte, _ := masterDomain.frc.h.BeginEndInterval()
		// 	fmt.Printf("   >>>>>> model will proceed from %s to %s (%d timesteps)\n", dtb.Format("2006-01-02"), dte.Format("2006-01-02"), masterDomain.frc.h.Nstep())
		// } else {
		// 	b = masterDomain.newSubDomain(loadForcing(metfp, true)) // gauge outlet cell id found in .met file
		// }
	}
	proceed = true
	return
}

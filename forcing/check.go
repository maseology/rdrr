package forcing

import "fmt"

func (frc *Forcing) CheckAndPrint() {
	fmt.Println("Forcing summary:")
	nt := len(frc.T)
	fmt.Printf(" %v to %v, 6-hourly (%d timesteps)\n", frc.T[0], frc.T[nt-1], nt)
	nsta := len(frc.Ya)
	fmt.Printf(" model timestep interval: %ds, %d stations\n", int64(frc.IntervalSec), nsta)

	sy, se := 0., 0.
	for i := 0; i < nsta; i++ {
		for j := range frc.T {
			sy += frc.Ya[i][j]
			se += frc.Ea[i][j]
		}
	}

	switch frc.IntervalSec {
	case 86400.:
		sy *= 365.24 / float64(nt) / float64(nsta)
		se *= 365.24 / float64(nt) / float64(nsta)
	case 86400. / 4:
		sy *= 365.24 * 4. / float64(nt) / float64(nsta)
		se *= 365.24 * 4. / float64(nt) / float64(nsta)
	default:
		panic(fmt.Sprintf("forcing.CheckAndPrint Error: timestep %f not recognized", frc.IntervalSec))
	}

	fmt.Printf(" totals (m/yr): Ya: %.5f   Ea: %.5f\n", sy, se)
}

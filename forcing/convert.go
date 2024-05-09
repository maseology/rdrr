package forcing

func (frc *Forcing) ToMM() {
	for j := range frc.T {
		for i := 0; i < len(frc.Ya); i++ {
			frc.Ya[i][j] *= 1000
		}
	}
	if len(frc.Ea) > 0 {
		for j := range frc.T {
			for i := 0; i < len(frc.Ya); i++ {
				frc.Ea[i][j] *= 1000
			}
		}
	}
}

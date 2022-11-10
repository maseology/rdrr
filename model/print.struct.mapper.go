package model

// Prints a set of rasters for verification
func (m *MAPR) PrintAndCheck(dir string, ocids []int) error {
	lui, sgi := make([]int32, len(ocids)), make([]int32, len(ocids))
	for i, c := range ocids {
		lui[i] = int32(m.LUx[c])
		sgi[i] = int32(m.SGx[c])
	}
	writeInts(dir+"/check/MAPR-LandUseID.indx", lui)
	writeInts(dir+"/check/MAPR-SurfGeoID.indx", sgi)

	return nil
}

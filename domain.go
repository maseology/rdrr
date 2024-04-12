package rdrr

import "github.com/maseology/goHydro/forcing"

func LoadDomain(mdlprfx string, cid0 int) (*Structure, *Subwatershed, *Mapper, *Parameter, *forcing.Forcing) {
	chkerr := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	strc, err := LoadGobStructure(mdlprfx + "structure.gob")
	chkerr(err)
	sws, err := loadGobSubwatershed(mdlprfx + "subwatershed.gob")
	chkerr(err)
	mp, err := LoadGobMapper(mdlprfx + "mapper.gob")
	chkerr(err)
	par, err := loadGobParameter(mdlprfx + "parameter.gob")
	chkerr(err)
	frc, err := forcing.LoadGobForcing(mdlprfx + "forcing.gob")
	chkerr(err)

	// if cid0 >= 0 {

	// }

	return strc, sws, mp, par, frc
}

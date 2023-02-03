package rdrr

func LoadDomain(mdlprfx string) (*Structure, *Subwatershed, *Mapper, *Parameter, *Forcing) {
	chkerr := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	strc, err := loadGobStructure(mdlprfx + "structure.gob")
	chkerr(err)
	sws, err := loadGobSubwatershed(mdlprfx + "subwatershed.gob")
	chkerr(err)
	mp, err := loadGobMapper(mdlprfx + "mapper.gob")
	chkerr(err)
	par, err := loadGobParameter(mdlprfx + "parameter.gob")
	chkerr(err)
	frc, err := loadGobForcing(mdlprfx + "forcing.gob")
	chkerr(err)

	return strc, sws, mp, par, frc
}

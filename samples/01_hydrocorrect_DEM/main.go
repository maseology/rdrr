package main

/*
	Regionally-Distributed Rainfall Runoff Recharge model
	version 0.1.1

    this example imports a regional hydrologically-"correct" digital elevation model
    inputs basin-wide average meterological imput and the cell ID of the observation point
*/

import (
	"fmt"
	"log"
	"sync"

	"github.com/maseology/goHydro/met"
	gwru "github.com/maseology/rdrr/gwru"
	hru "github.com/maseology/rdrr/hru"
	"github.com/maseology/rdrr/lusg"
	"github.com/maseology/rdrr/tem"
)

const (
	indir = "C:/Users/mason/Desktop/CAMC_5000/"
	metfp = "02EC018.met"
	temfp = "ORMGP_50_hydrocorrect.uhdem"
	lufp  = "ORMGP_50_LU.real"
	sgfp  = "ORMGP_50_SG.real"
)

func main() {
	frc, topo := load()
	for d := range frc {
		fmt.Println(d)
	}
	fmt.Println(topo.NumCells())
}

func load() (met.Coll, tem.TEM) {
	var wgVar, wgStrc sync.WaitGroup

	// variables, forcings, etc.
	var dc met.Coll
	readmet := func() {
		defer wgVar.Done()
		d, err := met.ReadMET(metfp)
		if err != nil {
			log.Fatalln(err)
		}
		dc = d
	}
	wgVar.Add(1)
	go readmet()

	// structural data
	var t tem.TEM
	var lu lusg.LandUse
	var sg lusg.SurfGeo
	readtopo := func() {
		defer wgStrc.Done()
		t.New(temfp)
	}
	readLU := func() {
		defer wgStrc.Done()
		lu.New(lufp)
	}
	readSG := func() {
		defer wgStrc.Done()
		sg.New(sgfp)
	}

	wgStrc.Add(3)
	go readtopo()
	go readLU()
	go readSG()

	// build hru's, and gw reservoir
	wgStrc.Wait()
	cid0 := 12778941 // outlet cid, get from .met ////////////////////////////////////////////////////////
	var b hru.Basin  //= make(make(map[int]hru.HRU, t.UpCnt(cid0)))
	var g gwru.GWmodel
	assignHRUs := func() {
		defer wgStrc.Done()
		ts := 6. * 60. * 60. // seconds get from .met //////////////////////////////////////////////

		var recurs func(int)
		recurs = func(cid int) {
			var h hru.HRU
			h.Initialize(1., 0., .2, 0.0001, ts) /////////////////////////////////////////////////////////////
			b[cid] = h
			for upcid := range t.UpIDs(cid) {
				recurs(upcid)
			}
		}
	}
	buildTopmodel := func() {
		defer wgStrc.Done()
		ksat := make(map[int]float64)
		var recurs func(int)
		recurs = func(cid int) {
			ksat[cid] = 0.0001
			for upcid := range t.UpIDs(cid) {
				recurs(upcid)
			}
		}
		recurs(cid0)
		g = &gwru.TOPMODEL{}
		g.New(ksat, t, 50., .1, 5., .1) /////////////////////////////////////////////////////////////
	}

	wgStrc.Add(2)
	go assignHRUs()
	go buildTopmodel()

	wgStrc.Wait()
	wgVar.Wait()
	return dc, t
}

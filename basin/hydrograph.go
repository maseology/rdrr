package basin

import "github.com/maseology/mmio"

// // ObsSim is used to create simple observed vs. simulated hydrographs
// func ObsSim(fp string, o, s, b, x []float64) {
// 	p, err := plot.New()
// 	if err != nil {
// 		panic(err)
// 	}

// 	// p.Title.Text = fp
// 	p.X.Label.Text = ""
// 	p.Y.Label.Text = "discharge"

// 	ps, err := plotter.NewLine(sequentialLine(s))
// 	if err != nil {
// 		panic(err)
// 	}
// 	ps.Color = color.RGBA{R: 255, A: 255}

// 	po, err := plotter.NewLine(sequentialLine(o))
// 	if err != nil {
// 		panic(err)
// 	}
// 	po.Color = color.RGBA{B: 255, A: 255}

// 	if b != nil {
// 		pb, err := plotter.NewLine(sequentialLine(b))
// 		if err != nil {
// 			panic(err)
// 		}
// 		pb.Color = color.RGBA{R: 128, B: 64, A: 255}
// 		px, err := plotter.NewLine(sequentialLine(x))
// 		if err != nil {
// 			panic(err)
// 		}
// 		pb.Color = color.RGBA{G: 128, B: 64, A: 255}
// 		// Add the functions and their legend entries.
// 		p.Add(ps, po, pb, px)
// 		p.Legend.Add("obs", po)
// 		p.Legend.Add("sim", ps)
// 		p.Legend.Add("bf", pb)
// 		p.Legend.Add("xs", px)
// 	} else {
// 		// Add the functions and their legend entries.
// 		p.Add(ps, po)
// 		p.Legend.Add("obs", po)
// 		p.Legend.Add("sim", ps)
// 	}
// 	p.Legend.Top = true
// 	// p.X.Tick.Marker = plot.TimeTicks{Format: "Jan"}

// 	// Save the plot to a PNG file.
// 	if err := p.Save(24*vg.Inch, 8*vg.Inch, fp); err != nil {
// 		panic(err)
// 	}
// }

func sumPlotSto(fp string, h, d []float64) {
	xs, ys := make([]float64, len(h)), make(map[string][]float64, 2)
	for i := range h {
		xs[i] = float64(i)
	}
	ys["sto"] = h
	ys["def"] = d
	mmio.Line(fp, xs, ys, 48.)
}

// func sumPlotHydrographWB(fp string, s, d, k, x, a, g []interface{}) {
// 	xs, ys := make([]float64, len(s)), make(map[string][]float64, 6)
// 	for i := range s {
// 		xs[i] = float64(i)
// 	}
// 	ys["sto"] = mmio.InterfaceToFloat(s)
// 	ys["def"] = mmio.InterfaceToFloat(d)
// 	ys["lag"] = mmio.InterfaceToFloat(k)
// 	ys["xs"] = mmio.InterfaceToFloat(x)
// 	ys["aet"] = mmio.InterfaceToFloat(a)
// 	ys["rch"] = mmio.InterfaceToFloat(g)
// 	mmio.Line(fp, xs, ys)
// }

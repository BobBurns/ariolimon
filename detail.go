package main

import (
	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/vg"
	"github.com/gonum/plot/vg/draw"
	"image/color"
	"time"
)

// returns last value and plots a graph
func graphMetric(metrics []QueryResult, title string) (QueryResult, error) {

	var trans float64 = 1.0
	var tlabel string = metrics[0].Units
	if tlabel == "Bytes" {
		for _, data := range metrics {
			if data.Value > 1048576.0 {
				trans = 1048576.0
				tlabel = "MB"
			} else if data.Value > 1028.0 {
				trans = 1028.0
				tlabel = "KB"
			}
		}
	}
	local := func(f float64) time.Time {
		return time.Unix(int64(f), 0).Local()
	}
	xticks := plot.TimeTicks{
		Format: time.Stamp,
		Time:   local,
	}
	pts := make(plotter.XYs, len(metrics))
	for i := range pts {
		pts[i].X = metrics[i].Time
		pts[i].Y = metrics[i].Value / trans
	}
	p, err := plot.New()
	if err != nil {
		return QueryResult{}, err
	}
	//	values = pts
	p.Title.Text = title
	p.X.Tick.Marker = xticks
	p.Y.Label.Text = tlabel
	p.Add(plotter.NewGrid())

	line, points, err := plotter.NewLinePoints(pts)
	if err != nil {
		return QueryResult{}, err
	}

	line.Color = color.RGBA{R: 17, G: 11, B: 192, A: 255}
	points.Shape = draw.CircleGlyph{}
	points.Color = color.RGBA{A: 255}

	p.Add(line, points)
	err = p.Save(20*vg.Centimeter, 10*vg.Centimeter, "html/currentgraph.png")
	if err != nil {
		return QueryResult{}, err
	}
	return metrics[len(metrics)-1], nil
}

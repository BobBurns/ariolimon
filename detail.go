package main

import (
	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/vg"
	"github.com/gonum/plot/vg/draw"
	"image/color"
)

// returns last value and plots a graph
func graphMetric(metrics []QueryResult, title string) (QueryResult, error) {

	xticks := plot.TimeTicks{Format: "02 Jan 06\n15:04 UTC"}
	pts := make(plotter.XYs, len(metrics))
	for i := range pts {
		pts[i].X = metrics[i].Time
		pts[i].Y = metrics[i].Value
	}
	p, err := plot.New()
	if err != nil {
		return QueryResult{}, err
	}
	//	values = pts
	p.Title.Text = title
	p.X.Tick.Marker = xticks
	p.Y.Label.Text = title
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

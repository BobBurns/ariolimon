package main

import (
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/vg"
	"github.com/gonum/plot/vg/draw"
	"image/color"
	"sort"
)

type graphValues struct {
	Time  float64
	Value float64
	Unit  string
}
type ByTime []graphValues

func (a ByTime) Len() int {
	return len(a)
}
func (a ByTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ByTime) Less(i, j int) bool {
	return a[i].Time < a[j].Time
}

// returns last value and plots a graph
func graphMetric(metric *cloudwatch.GetMetricStatisticsOutput) (float64, string, error) {

	var gv = []graphValues{}
	for _, d := range metric.Datapoints {
		timeconv := *d.Timestamp
		unit, value := convertMetric(*d.Unit, *d.Maximum)
		values := graphValues{
			Time:  float64(timeconv.Unix()),
			Value: value,
			Unit:  unit,
		}
		gv = append(gv, values)
	}
	sort.Sort(ByTime(gv))
	xticks := plot.TimeTicks{Format: "02 Jan 06\n15:04 UTC"}
	pts := make(plotter.XYs, len(gv))
	for i := range pts {
		pts[i].X = gv[i].Time
		pts[i].Y = gv[i].Value
	}
	p, err := plot.New()
	if err != nil {
		return 0, "", err
	}
	//	values = pts
	p.Title.Text = *metric.Label
	p.X.Tick.Marker = xticks
	p.Y.Label.Text = "Max " + gv[0].Unit
	p.Add(plotter.NewGrid())

	line, points, err := plotter.NewLinePoints(pts)
	if err != nil {
		return 0, "", err
	}

	line.Color = color.RGBA{R: 17, G: 11, B: 192, A: 255}
	points.Shape = draw.CircleGlyph{}
	points.Color = color.RGBA{A: 255}

	p.Add(line, points)
	err = p.Save(20*vg.Centimeter, 10*vg.Centimeter, "html/currentgraph.png")
	if err != nil {
		return 0, "", err
	}
	return gv[len(gv)-1].Value, gv[len(gv)-1].Unit, nil
}

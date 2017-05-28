package main

import (
	"bytes"
	"fmt"
	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/vg"
	"github.com/gonum/plot/vg/draw"
	"github.com/gorilla/mux"
	"image/color"
	"log"
	"net/http"
	"regexp"
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

// custom detail from db

// detail handler
func detailHandler(w http.ResponseWriter, r *http.Request) {

	// map servicenames for query
	var hosts []MetricQuery
	err, hosts := getThresholds()
	if err != nil {
		log.Printf("Error with thresh.json: %s\n", err)
		http.Redirect(w, r, "/html/error.html", http.StatusFound)
	}
	var namemap = make(map[string]MetricQuery)
	for _, host := range hosts {
		namemap[host.Name] = host
	}

	vars := mux.Vars(r)
	query := vars["sd"]

	hostquery, ok := namemap[query]
	if ok == false {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		panic(err)
	}
	timeframe := r.FormValue("t")
	if timeframe == "" {
		timeframe = "-4h"
	}
	match, err := regexp.MatchString("^(-1h|-4h|-24h|-168h)$", timeframe)
	if match == false {
		http.Redirect(w, r, "/html/error.html", http.StatusFound)
		return
	}

	err = hostquery.getStatistics(timeframe)
	if err != nil {
		log.Printf("Error with getStatistics: %s", err)
		http.Redirect(w, r, "/html/error.html", http.StatusFound)
		return
	}

	title := ""
	fmt.Sprintf(title, "Statistics for %s", hostquery.Label)
	currentMetric, err := graphMetric(hostquery.Results, title)
	if err != nil {
		fmt.Fprintf(w, "%q\n", err)
	}
	detail := Detail{
		Host:    hostquery.Name,
		Service: hostquery.Label,
		Time:    time.Unix(int64(currentMetric.Time), 0).Format(time.RFC822),
		Alert:   currentMetric.Alert,
		Value:   currentMetric.Value,
		Units:   currentMetric.Units,
	}
	var b bytes.Buffer
	err = t.ExecuteTemplate(&b, "detail.html", detail)
	if err != nil {
		fmt.Fprintf(w, "Error with template: %s ", err)
		return
	}
	b.WriteTo(w)

}

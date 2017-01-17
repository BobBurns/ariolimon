package main

import (
	"bytes"
	"fmt"
	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/vg"
	"github.com/gonum/plot/vg/draw"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2/bson"
	"image/color"
	"log"
	"net/http"
	"regexp"
	"sort"
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
func customHandler(service Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := webSession(w, r)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
		var results []QueryStore
		const dateForm = "02 Jan 06 15:04"

		r.ParseForm()
		// check if form is posted with timeframe and metric query info
		servicePost := r.FormValue("service")
		if servicePost == "" {

			// if no form don't display metric image
			var b bytes.Buffer
			err := t.ExecuteTemplate(&b, "custom.html", service)
			if err != nil {
				fmt.Fprintf(w, "Error with template: %s ", err)
				return
			}
			b.WriteTo(w)
			return
		}

		// parse time info
		startTimeStr := r.FormValue("start_date")
		endTimeStr := r.FormValue("end_date")
		loc, _ := time.LoadLocation("Local")
		startTime, _ := time.ParseInLocation(dateForm, startTimeStr, loc)
		endTime, _ := time.ParseInLocation(dateForm, endTimeStr, loc)
		from := startTime.Unix()
		to := endTime.Unix()

		err = mcoll.Find(bson.M{
			"$and": []bson.M{bson.M{"uniquename": servicePost},
				bson.M{"unixtime": bson.M{
					"$gt": from,
					"$lt": to,
				}}}}).All(&results)
		if err != nil {
			http.Redirect(w, r, "/html/error.html", http.StatusFound)
			return
		}
		if len(results) == 0 {
			http.Redirect(w, r, "/html/nodata.html", http.StatusFound)
			return
		}

		var statistics []QueryResult
		for _, result := range results {
			qr := QueryResult{
				Units: result.Unit,
				Value: result.Value,
				Time:  result.UnixTime,
			}
			statistics = append(statistics, qr)
		}
		sort.Sort(ByTime(statistics))

		title := "Statistics for " + servicePost
		_, err = graphMetric(statistics, title)
		if err != nil {
			fmt.Fprintf(w, "%q\n", err)
		}

		var b bytes.Buffer
		err = t.ExecuteTemplate(&b, "custom-img.html", service)
		if err != nil {
			fmt.Fprintf(w, "Error with template: %s ", err)
			return
		}
		b.WriteTo(w)

	}
}

// detail handler
func detailHandler(hosts map[string]MetricQuery) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := webSession(w, r)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
		}

		vars := mux.Vars(r)
		query := vars["sd"]

		hostquery, ok := hosts[query]
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
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"time"
)

var t *template.Template

func init() {
	// map functions for http templates
	funcMap := template.FuncMap{
		"alertText": alertText,
		"ctime":     ctime,
	}
	// parse html template and threshold configuration file

	t = template.Must(template.New("templates").Funcs(funcMap).ParseFiles("html/templates/home2.html", "html/templates/detail.html", "html/templates/custom.html", "html/templates/custom-img.html"))

	// init cloudwatch session

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)
	if err != nil {
		panic(err)
	}

	svc = cloudwatch.New(sess)
	svc_ec2 = ec2.New(sess)

	// init mongo db
	msess, err = mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
	}
	msess.SetMode(mgo.Monotonic, true)

	mcoll = msess.DB("aws_metric_store").C("metric_values")
	index := mgo.Index{
		Key:        []string{"unixtime", "uniquename"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	err = mcoll.EnsureIndex(index)
	if err != nil {
		log.Fatalf("ensure index: %v", err)
	}
	// Parse config file
	data, err := ioutil.ReadFile("thresh.json")
	if err != nil {
		log.Fatalf("readfile: %v", err)
	}
	err = json.Unmarshal([]byte(data), &hosts)
	if err != nil {
		log.Fatalf("unmarshal: %v", err)
	}
	if debug == 1 {
		fmt.Println(hosts)
	}
}

// http handlers
func devHandler(querys []MetricQuery) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for i, _ := range querys {

			err := querys[i].getStatistics("-10m")
			if err != nil {
				log.Printf("Error with getStatistics: %s", err)
				http.Redirect(w, r, "/error", http.StatusFound)

			}
		}
		if debug == 1 {
			fmt.Printf("%v", querys)
		}
		var b bytes.Buffer
		err := t.ExecuteTemplate(&b, "home2.html", querys)
		if err != nil {
			fmt.Fprintf(w, "Error with template: %s ", err)
			return
		}
		b.WriteTo(w)

	}
}

func customHandler(service Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		err := mcoll.Find(bson.M{
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

func detailHandler(hosts map[string]MetricQuery) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

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

func main() {

	defer msess.Close()

	// map servicenames and hostnames for http handlers
	templateService := Services{}
	var namemap = make(map[string]MetricQuery)
	for i, _ := range hosts {
		namemap[hosts[i].Name] = hosts[i]
		templateService.Service = append(templateService.Service, hosts[i].Name)
	}

	router := mux.NewRouter()
	sub := router.Host("localhost").Subrouter()
	sub.PathPrefix("/html/").Handler(http.StripPrefix("/html/", http.FileServer(http.Dir("html"))))
	sub.HandleFunc("/", devHandler(hosts))
	sub.HandleFunc("/detail/{sd:[a-zA-Z0-9_-]+}", detailHandler(namemap))
	sub.HandleFunc("/custom", customHandler(templateService))

	server := http.Server{
		Addr:         ":8082",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      router,
	}
	fmt.Println("Server started at localhost:8082")
	log.Fatal(server.ListenAndServe())

}

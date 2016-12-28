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
	"sort"
	"time"
)

var t *template.Template

type Detail struct {
	Host    string
	Time    string
	Service string
	Alert   string
	Value   float64
	Units   string
}

func init() {
	// parse html template and threshold configuration file
	funcMap := template.FuncMap{
		"alertText": alertText,
		"ctime":     ctime,
	}

	t = template.Must(template.New("templates").Funcs(funcMap).ParseFiles("html/templates/home2.html", "html/templates/detail.html", "html/templates/root.html"))

	// init cloudwatch session

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)
	if err != nil {
		panic(err)
	}

	svc = cloudwatch.New(sess)
	svc_ec2 = ec2.New(sess)

	msess, err = mgo.Dial("127.0.0.1")
	if err != nil {
		panic(err)
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
func errHandler(w http.ResponseWriter, r *http.Request) {
	//write error mess
	fmt.Fprintf(w, "Oops! Internal Error.\nNo Data Available.\n****************")
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	//write error mess

	//	log.Fatal("not found")
	http.Redirect(w, r, "/device/", http.StatusFound)
}
func customHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fmt.Println("service: ", r.Form["service"])
	fmt.Println("start date: ", r.Form["start_date"])
	fmt.Println("end date: ", r.Form["end_date"])
	fmt.Println("start time: ", r.Form["start_time"])
	fmt.Println("end time: ", r.Form["end_time"])
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	var results []QueryStore

	now := time.Now()
	to := float64(now.Unix())

	duration, _ := time.ParseDuration("-2h")
	s := now.Add(duration)
	from := float64(s.Unix())
	err := mcoll.Find(bson.M{
		"$and": []bson.M{bson.M{"uniquename": "apm1_cpuu"},
			bson.M{"unixtime": bson.M{
				"$gt": from,
				"$lt": to,
			}}}}).All(&results)
	if err != nil {
		panic(err)
	}

	//	fmt.Printf("Results: %v\n", results)
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

	title := "Statistics for apm1_cpuu"
	currentMetric, err := graphMetric(statistics, title)
	if err != nil {
		fmt.Fprintf(w, "%q\n", err)
	}
	detail := Detail{
		Host:    "apm1",
		Service: "cpu-utilization",
		Time:    time.Unix(int64(currentMetric.Time), 0).Format(time.RFC822),
		Alert:   "info",
		Value:   currentMetric.Value,
		Units:   currentMetric.Units,
	}
	//currentMetric.compareThresh(hostquery.Warning, hostquery.Critical)
	var b bytes.Buffer
	err = t.ExecuteTemplate(&b, "detail.html", detail)
	if err != nil {
		fmt.Fprintf(w, "Error with template: %s ", err)
		return
	}
	b.WriteTo(w)

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
		fmt.Println("time ", timeframe)
		/*
			switch timeframe {
			case "4h":
				timeframe = "-4h"
			case "24h":
				timeframe = "-24h"
			default:
				timeframe = "-4h"
			}
		*/

		err := hostquery.getStatistics(timeframe)
		if err != nil {
			log.Printf("Error with getStatistics: %s", err)
			http.Redirect(w, r, "/error", http.StatusFound)
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
		//currentMetric.compareThresh(hostquery.Warning, hostquery.Critical)
		var b bytes.Buffer
		err = t.ExecuteTemplate(&b, "detail.html", detail)
		if err != nil {
			fmt.Fprintf(w, "Error with template: %s ", err)
			return
		}
		b.WriteTo(w)

	}
}

// function to handle template output .Alert text
func alertText(alert string) string {
	switch alert {
	case "danger":
		return "Critical"
	case "warning":
		return "Warning"
	case "success":
		return "OK"
	case "info":
		return "Unknown"
	}

	return "Unknown"
}
func ctime() string {
	return time.Now().Format(time.RFC822)
}

func main() {

	defer msess.Close()
	// initialize database (move to init())
	msess.SetMode(mgo.Monotonic, true)

	mcoll = msess.DB("aws_metric_store").C("metric_values")
	index := mgo.Index{
		Key:        []string{"unixtime", "uniquename"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	err := mcoll.EnsureIndex(index)
	if err != nil {
		log.Fatalf("ensure index: %v", err)
	}

	// Parse config file
	var hosts []MetricQuery
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
	// make a map of hostnames to MetricQuery
	var namemap = make(map[string]MetricQuery)
	for i, _ := range hosts {
		namemap[hosts[i].Name] = hosts[i]
	}

	router := mux.NewRouter()
	sub := router.Host("localhost").Subrouter()
	sub.PathPrefix("/html/").Handler(http.StripPrefix("/html/", http.FileServer(http.Dir("html"))))
	//	sub.PathPrefix("/device/html/").Handler(http.StripPrefix("/device/html/", http.FileServer(http.Dir("html"))))

	sub.HandleFunc("/", devHandler(hosts))
	sub.HandleFunc("/detail/{sd:[a-zA-Z0-9_-]+}", detailHandler(namemap))
	sub.HandleFunc("/error", errHandler)
	sub.HandleFunc("/test", testHandler)
	sub.HandleFunc("/custom", customHandler)

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

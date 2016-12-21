package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
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
		fmt.Println("falied to create session,", err)
		return
	}

	svc = cloudwatch.New(sess)
	svc_ec2 = ec2.New(sess)
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

func detailHandler(hosts map[string]MetricQuery) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//		err := validatePath(w, r)
		//		if err != nil {
		//			return
		//		}
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "Error with ParseForm\n")
			return
		}
		query := r.FormValue("q")

		if query == "" {
			http.Redirect(w, r, "/device/", http.StatusFound)
			return
		}
		if debug == 2 {
			fmt.Printf("Query = %s", query)
		}

		hostquery := hosts[query]
		err := hostquery.getStatistics("-4h")
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
			Host:    hostquery.Host,
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
func validatePath(w http.ResponseWriter, r *http.Request) error {
	fmt.Println(r.URL.Path)
	var validPath = regexp.MustCompile("^/device/detail?q=[a-zA-Z0-9_-]+$")
	if validPath.MatchString(r.URL.Path) {
		return nil
	}
	http.NotFound(w, r)
	return errors.New("Invalid Page Title")
}

func main() {

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
	// TODO: handle file directory html/random

	mux := http.NewServeMux()
	mux.Handle("/html/", http.StripPrefix("/html/", http.FileServer(http.Dir("html"))))
	mux.Handle("/device/html/", http.StripPrefix("/device/html/", http.FileServer(http.Dir("html"))))
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/device/", devHandler(hosts))
	mux.HandleFunc("/device/detail", detailHandler(namemap))
	mux.HandleFunc("/error", errHandler)

	server := http.Server{
		Addr:         ":8082",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      mux,
	}
	fmt.Println("Server started at localhost:8082")
	server.ListenAndServe()

}

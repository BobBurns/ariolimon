package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
)

var t *template.Template

func init() {
	// parse html template and threshold configuration file
	funcMap := template.FuncMap{
		"alertText": alertText,
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

	http.Redirect(w, r, "/device/", http.StatusFound)
}

func detailHandler(hosts map[string]MetricQuery) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "Error with ParseForm\n")
			return
		}
		query := r.FormValue("q")

		if query == "" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		hostquery := hosts[query]
		err := hostquery.getStatistics("-4h")
		if err != nil {
			log.Fatalf("Error with getMetricDetail: %s", err)
		}

		title := ""
		fmt.Sprintf(title, "%s %s", hostquery.Statistics, hostquery.Results[0].Units)
		currentMetric, err := graphMetric(hostquery.Results, title)
		if err != nil {
			fmt.Fprintf(w, "%q\n", err)
		}
		currentMetric.compareThresh(hostquery.Warning, hostquery.Critical)
		var b bytes.Buffer
		err = t.ExecuteTemplate(&b, "detail.html", currentMetric)
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
	}

	return "Unknown"
}

func main() {
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
	http.Handle("/html/", http.StripPrefix("/html/", http.FileServer(http.Dir("html"))))
	http.Handle("/device/html/", http.StripPrefix("/device/html/", http.FileServer(http.Dir("html"))))
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/device/", devHandler(hosts))
	http.HandleFunc("/device/detail", detailHandler(namemap))
	http.HandleFunc("/error", errHandler)
	http.ListenAndServe(":8082", nil)
}

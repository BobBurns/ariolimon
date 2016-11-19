package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"html/template"
	"net/http"
	"os"
	"strings"
)

var t *template.Template
var thresh = make(map[string][]string)

func init() {
	// parse html template and threshold configuration file
	funcMap := template.FuncMap{
		"alertText": alertText,
	}

	t = template.Must(template.New("templates").Funcs(funcMap).ParseFiles("html/templates/home2.html", "html/templates/detail.html"))

	f, err := os.Open("thresholds.conf")
	if err != nil {
		fmt.Printf("Could not open thresholds.conf: %s", err)
		os.Exit(1)
	}
	// map metric names and thresholds
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		result := strings.Fields(scanner.Text())
		thresh[result[0]] = []string{result[1], result[2]}
	}
	// init cloudwatch session

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)
	if err != nil {
		fmt.Println("falied to create session,", err)
		return
	}

	svc = cloudwatch.New(sess)
}

// http handlers

func rootHandler(w http.ResponseWriter, r *http.Request) {
	// get aws metric data
	q, err := statQuery()
	if err != nil {
		fmt.Fprintf(w, "Error with statQuery: %s", err)
		os.Exit(1)
	}

	var b bytes.Buffer
	err = t.ExecuteTemplate(&b, "home2.html", q)
	if err != nil {
		fmt.Fprintf(w, "Error with template: %s ", err)
		return
	}
	b.WriteTo(w)
}

func detailHandler(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "Error with ParseForm\n")
		return
	}
	// sanity check
	q := r.FormValue("q")
	if q == "" || base == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		//		fmt.Fprintf(w, "Malformed query!\n")
		return
	}
	resp, err := getMetricDetail(q, "4 hours")
	if err != nil {
		fmt.Fprintf(w, "%q\n", err)
	}
	value, err := graphMetric(resp)
	if err != nil {
		fmt.Fprintf(w, "%q\n", err)
	}
	currentMetric := Metric{
		Label:      *resp.Label,
		Units:      *resp.Datapoints[0].Unit,
		Statistics: "Average",
		Value:      value,
	}

	currentMetric.Alert = compareThresh(currentMetric)

	var b bytes.Buffer
	err = t.ExecuteTemplate(&b, "detail.html", currentMetric)
	if err != nil {
		fmt.Fprintf(w, "Error with template: %s ", err)
		return
	}
	b.WriteTo(w)

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
	// TODO: handle file directory html/random
	http.Handle("/html/", http.StripPrefix("/html/", http.FileServer(http.Dir("html"))))
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/detail", detailHandler)
	http.ListenAndServe(":8082", nil)
}

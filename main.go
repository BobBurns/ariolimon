package main

import (
	"bufio"
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

	t = template.Must(template.New("templates").Funcs(funcMap).ParseFiles("html/templates/home2.html", "html/templates/detail.html", "html/templates/root.html"))

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
	svc_ec2 = ec2.New(sess)
}

// http handlers
func devHandler(hosts map[string]EC2MetricsQuery) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hostname := r.URL.Path[len("/device/"):]
		hostquerry := hosts[hostname]
		fmt.Println(hosts[hostname])
		err := hostquerry.getStatistics()
		if err != nil {
			log.Printf("Error with getStatistics: %s", err)
			http.Redirect(w, r, "/error", http.StatusFound)

		}
		var b bytes.Buffer
		err = t.ExecuteTemplate(&b, "home2.html", hostquerry)
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

func rootHandler(hostnames []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var b bytes.Buffer
		err := t.ExecuteTemplate(&b, "root.html", hostnames)
		if err != nil {
			fmt.Fprintf(w, "Error with template: %s ", err)
			return
		}
		b.WriteTo(w)

	}
}

func detailHandler(hosts map[string]EC2MetricsQuery) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "Error with ParseForm\n")
			return
		}
		query := r.FormValue("q")
		host := r.FormValue("host")
		stat := r.FormValue("stat")

		if query == "" || host == "" {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		hostquery := hosts[host]
		fmt.Println(hosts[host])
		results, err := hostquery.getMetricDetail(stat, query, "4 hours")
		if err != nil {
			log.Fatalf("Error with getMetricDetail: %s", err)
		}

		currentMetric, err := graphMetric(results)
		if err != nil {
			fmt.Fprintf(w, "%q\n", err)
		}
		currentMetric.compareThresh()
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
	var hosts []EC2MetricsQuery
	data, err := ioutil.ReadFile("thresh.json")
	if err != nil {
		log.Fatalf("readfile: %v", err)
	}
	err = json.Unmarshal([]byte(data), &hosts)
	if err != nil {
		log.Fatalf("unmarshal: %v", err)
	}
	fmt.Println(hosts)
	// make a map of hostnames to EC2MetricsQuery
	var hostmap = make(map[string]EC2MetricsQuery)
	var hostnames []string
	for _, query := range hosts {
		hostmap[query.Host] = query
		hostnames = append(hostnames, query.Host)
	}
	// TODO: handle file directory html/random
	http.Handle("/html/", http.StripPrefix("/html/", http.FileServer(http.Dir("html"))))
	http.Handle("/device/html/", http.StripPrefix("/device/html/", http.FileServer(http.Dir("html"))))
	http.HandleFunc("/", rootHandler(hostnames))
	http.HandleFunc("/device/", devHandler(hostmap))
	http.HandleFunc("/device/detail", detailHandler(hostmap))
	http.HandleFunc("/error", errHandler)
	http.ListenAndServe(":8082", nil)
}

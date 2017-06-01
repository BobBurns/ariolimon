package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var t *template.Template
var svc *cloudwatch.CloudWatch
var svc_ec2 *ec2.EC2

//var msess *mgo.Session

func init() {
	// map functions for http templates
	funcMap := template.FuncMap{
		"alertText": alertText,
		"ctime":     ctime,
	}

	// parse html template and threshold configuration file
	t = template.Must(template.New("templates").Funcs(funcMap).ParseFiles("html/templates/crit.html", "html/templates/detail.html", "html/templates/warn.html", "html/templates/ok.html", "html/templates/home2.html"))

	// init cloudwatch session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)
	if err != nil {
		panic(err)
	}

	svc = cloudwatch.New(sess)
	svc_ec2 = ec2.New(sess)

}

// sort functions
type ByLabel []MetricQuery
type ByTime []QueryResult

func (a ByTime) Len() int {
	return len(a)
}
func (a ByTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ByTime) Less(i, j int) bool {
	return a[i].Time < a[j].Time
}

func (a ByLabel) Len() int {
	return len(a)
}

func (a ByLabel) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByLabel) Less(i, j int) bool {
	return a[i].Label < a[j].Label
}

//helper for aws time period
func getPeriod(time string) (period int64) {
	period = 360
	switch time {
	case "-4h":
		period = 600
	case "-24h":
		period = 3600
	case "-168h":
		period = 14400
	}
	return
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

//template function
func ctime() string {
	return time.Now().Format(time.RFC822)
}

// function to dynamically load thresholds
func getThresholds() (error, []MetricQuery) {
	var hosts []MetricQuery
	// Parse threshold file
	data, err := ioutil.ReadFile("thresh.json")
	if err != nil {
		return err, nil
	}
	err = json.Unmarshal([]byte(data), &hosts)
	if err != nil {
		return err, nil
	}
	if debug == 1 {
		fmt.Println(hosts)
	}
	return nil, hosts
}

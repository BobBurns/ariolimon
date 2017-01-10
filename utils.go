package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"gopkg.in/mgo.v2"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var t *template.Template
var svc *cloudwatch.CloudWatch
var svc_ec2 *ec2.EC2
var msess *mgo.Session
var mcoll *mgo.Collection
var dbcoll *mgo.Collection
var hosts []MetricQuery

func init() {
	// map functions for http templates
	funcMap := template.FuncMap{
		"alertText": alertText,
		"ctime":     ctime,
	}

	// parse html template and threshold configuration file
	t = template.Must(template.New("templates").Funcs(funcMap).ParseFiles("html/templates/home2.html", "html/templates/detail.html", "html/templates/custom.html", "html/templates/custom-img.html", "html/templates/login.html"))

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
	// get db config
	dbData, err := os.Open("configdb.json")
	if err != nil {
		log.Fatalf("open configdb: %v", err)
	}
	configdb := struct {
		Host string
		User string
		Pass string
		Db   string
	}{}

	decoder := json.NewDecoder(dbData)
	err = decoder.Decode(&configdb)
	if err != nil {
		log.Fatalf("decode: %v", err)
	}
	dburl := configdb.User + ":" + configdb.Pass + "@" + configdb.Host + "/" + configdb.Db

	msess, err = mgo.Dial(dburl)
	if err != nil {
		panic(err)
	}
	msess.SetMode(mgo.Monotonic, true)

	// connection to metric_values
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

	// set up connection to user collection
	dbcoll = msess.DB("aws_metric_store").C("aws_usr")
	usrindex := mgo.Index{
		Key:        []string{"name"},
		Unique:     true,
		DropDups:   false,
		Background: true,
		Sparse:     true,
	}
	err = dbcoll.EnsureIndex(usrindex)
	if err != nil {
		log.Fatalf("ensure index: %v", err)
	}

	// Parse threshold file
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

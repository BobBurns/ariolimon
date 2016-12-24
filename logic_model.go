package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"sort"
	"strconv"
	"strings"
	"time"
)

const debug int = 2

var svc *cloudwatch.CloudWatch
var svc_ec2 *ec2.EC2
var msess *mgo.Session
var mcoll *mgo.Collection

type Dimension struct {
	DimName  string `json:"dim_name"`
	DimValue string `json:"dim_value"`
}
type QueryResult struct {
	Alert string
	Units string
	Value float64
	Time  float64
}
type QueryStore struct {
	ID         bson.ObjectId `bson:"_id,omitempty"`
	UniqueName string
	Value      float64
	Unit       string
	UnixTime   float64
}

type MetricQuery struct {
	Name       string      `json:"name"`
	Host       string      `json:"hostname"`
	Namespace  string      `json:"namespace"`
	Dims       []Dimension `json:"dimensions"`
	Label      string      `json:"metric"`
	Statistics string      `json:"statistics"`
	Warning    string      `json:"warning"`
	Critical   string      `json:"critical"`
	Results    []QueryResult
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

func (mq *MetricQuery) getStatistics(timeframe string) error {

	t := time.Now()
	if mq.Namespace == "AWS/S3" {
		timeframe = "-36h"
	}
	duration, _ := time.ParseDuration(timeframe)
	s := t.Add(duration)
	var dims []*cloudwatch.Dimension
	for i := 0; i < len(mq.Dims); i++ {
		dims = append(dims, &cloudwatch.Dimension{
			Name:  aws.String(mq.Dims[i].DimName),
			Value: aws.String(mq.Dims[i].DimValue),
		})
	}
	params := cloudwatch.GetMetricStatisticsInput{
		EndTime:    aws.Time(t),
		Namespace:  aws.String(mq.Namespace),
		Period:     aws.Int64(360),
		StartTime:  aws.Time(s),
		Dimensions: dims,
		MetricName: aws.String(mq.Label),
		Statistics: []*string{
			aws.String(mq.Statistics),
		},
	}
	resp, err := svc.GetMetricStatistics(&params)
	if err != nil {
		return fmt.Errorf("Metric query failed: %s", err.Error())
	}
	if len(resp.Datapoints) == 0 {
		if debug == 1 {
			fmt.Println("no datapoints")
		}

		data := QueryResult{
			Value: 0.0,
			Units: "Unknown",
			Time:  float64(time.Now().Unix()),
			Alert: "info",
		}
		mq.Results = append(mq.Results, data)
		return nil
	}
	for _, dp := range resp.Datapoints {
		unit := *dp.Unit
		value := 0.0
		switch mq.Statistics {
		case "Maximum":
			value = *dp.Maximum
		case "Average":
			value = *dp.Average
		case "Sum":
			value = *dp.Sum
		case "SampleCount":
			value = *dp.SampleCount
		case "Minimum":
			value = *dp.Minimum
		}

		if unit == "Bytes" {
			if value > 1048576.0 {
				value = value / 104857.0
				unit = "MB"
			} else if value > 1028.0 {
				value = value / 1028.0
				unit = "KB"
			}
		}
		data := QueryResult{
			Value: value,
			Units: unit,
			Time:  float64(dp.Timestamp.Unix()),
		}
		data.compareThresh(mq.Warning, mq.Critical)
		mq.Results = append(mq.Results, data)
	}

	sort.Sort(ByTime(mq.Results))
	if debug == 1 {
		fmt.Printf("Get Statistics Result: %v", mq)
	}
	// persist result
	for _, qr := range mq.Results {
		data := QueryStore{
			UniqueName: mq.Name,
			Value:      qr.Value,
			Unit:       qr.Units,
			UnixTime:   qr.Time,
		}
		err = mcoll.Insert(&data)
		if err != nil {
			panic(err)
		}
	}

	return nil
}

// function to compare threshold with query values and return html ready warning

func (qr *QueryResult) compareThresh(warn, crit string) {
	// adjust for transform
	value := qr.Value // make a copy

	if qr.Units == "MB" {
		value = value * 1048576.0
	}
	if qr.Units == "KB" {
		value = value * 1024.0
	}

	var minwarn float64 = 0.0
	var maxwarn float64 = 100.0
	var mincrit float64 = 0.0
	var maxcrit float64 = 100.0
	warnings := strings.Split(warn, ":")
	if len(warnings) < 2 {
		minwarn = 0
		maxwarn, _ = strconv.ParseFloat(warnings[0], 64)
	} else {
		minwarn, _ = strconv.ParseFloat(warnings[0], 64)
		maxwarn, _ = strconv.ParseFloat(warnings[1], 64)
	}
	criticals := strings.Split(crit, ":")
	if len(criticals) < 2 {
		mincrit = 0.0
		maxcrit, _ = strconv.ParseFloat(criticals[0], 64)
	} else {
		mincrit, _ = strconv.ParseFloat(criticals[0], 64)
		maxcrit, _ = strconv.ParseFloat(criticals[1], 64)
	}
	qr.Alert = "success"
	if value > maxcrit || value < mincrit {
		qr.Alert = "danger"
	} else if value > maxwarn || value < minwarn {
		qr.Alert = "warning"
	}

}

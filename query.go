package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"sort"
	"strconv"
	"strings"
	"time"
)

const debug int = 0

// Detail used to display last data
// in detail.html
type Detail struct {
	Host    string
	Time    string
	Service string
	Alert   string
	Value   float64
	Units   string
}

// QueryResult populated after call to
// cloudwatch.GetMetricStatistics
type QueryResult struct {

	// alert value for display
	Alert string

	// unit type eg Bytes, Count, etc
	Units string

	// query result value
	Value float64

	// Unix time converted to float64
	Time float64
}

type MetricQuery struct {
	// populated from thresh.json
	Name       string      `json:"name"`
	Host       string      `json:"hostname"`
	Namespace  string      `json:"namespace"`
	Dims       []Dimension `json:"dimensions"`
	Label      string      `json:"metric"`
	Statistics string      `json:"statistics"`
	Warning    string      `json:"warning"`
	Critical   string      `json:"critical"`

	// results from aws query
	Results []QueryResult
}
type Dimension struct {
	DimName  string `json:"dim_name"`
	DimValue string `json:"dim_value"`
}

// getStatistics called after MetricQuery parameters are loaded with getThresholds
// returns with MetricQuery.Results poplulated with cloudwatch data
// will error if input values are incorrect or missing
func (mq *MetricQuery) getStatistics(timeframe string) error {

	// convert timeframe into minutes
	period := getPeriod(timeframe)
	t := time.Now()
	if mq.Namespace == "AWS/S3" {
		timeframe = "-36h"
	}
	duration, _ := time.ParseDuration(timeframe)
	s := t.Add(duration)
	var dims []*cloudwatch.Dimension

	// handle multiple dimensions
	for i := 0; i < len(mq.Dims); i++ {
		dims = append(dims, &cloudwatch.Dimension{
			Name:  aws.String(mq.Dims[i].DimName),
			Value: aws.String(mq.Dims[i].DimValue),
		})
	}

	// fill out GetMetricStatisticsInput for aws query
	params := cloudwatch.GetMetricStatisticsInput{
		EndTime:    aws.Time(t),
		Namespace:  aws.String(mq.Namespace),
		Period:     aws.Int64(period),
		StartTime:  aws.Time(s),
		Dimensions: dims,
		MetricName: aws.String(mq.Label),
		Statistics: []*string{
			aws.String(mq.Statistics),
		},
	}

	// aws cloudwatch call
	resp, err := svc.GetMetricStatistics(&params)
	if err != nil {
		return fmt.Errorf("Metric query failed: %s", err.Error())
	}

	// handle no data returned from query
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

	// iterate through datapoints and append to MetricQuery.Results array
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

		data := QueryResult{
			Value: value,
			Units: unit,
			Time:  float64(dp.Timestamp.Unix()),
		}
		data.compareThresh(mq.Warning, mq.Critical)
		mq.Results = append(mq.Results, data)
	}

	// sort data points by time
	sort.Sort(ByTime(mq.Results))
	if debug == 1 {
		fmt.Printf("Get Statistics Result: %v", mq)
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

	// alerts for pretty twitter bootstrap colors
	qr.Alert = "success"
	if value > maxcrit || value < mincrit {
		qr.Alert = "danger"
	} else if value > maxwarn || value < minwarn {
		qr.Alert = "warning"
	}

}

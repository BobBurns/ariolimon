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

var svc *cloudwatch.CloudWatch
var base *MetricBaseParams

type MetricBaseParams struct {
	DimName   string
	DimValue  string
	Namespace string
}

type EC2MetricsQuery struct {
	TotalCount int
	Time       string
	Items      []Metric
}

type Metric struct {
	Label      string
	Units      string
	Statistics string
	Alert      string
	Value      float64
}

type ByLabel []Metric

func (a ByLabel) Len() int {
	return len(a)
}

func (a ByLabel) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByLabel) Less(i, j int) bool {
	return a[i].Label < a[j].Label
}

func getStatistics(metrics []string) (*EC2MetricsQuery, error) {

	mq := EC2MetricsQuery{
		TotalCount: len(metrics),
		Time:       time.Now().Format(time.RFC822),
	}
	t := time.Now()
	duration, _ := time.ParseDuration("-10m")
	s := t.Add(duration)
	params := cloudwatch.GetMetricStatisticsInput{
		EndTime:   aws.Time(t),
		Namespace: aws.String(base.Namespace),
		Period:    aws.Int64(300),
		//		MetricName: aws.String(metric),
		StartTime: aws.Time(s),
		Statistics: []*string{
			aws.String("Average"),
		},
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String(base.DimName),
				Value: aws.String(base.DimValue),
			},
		},
	}
	for _, metric := range metrics {
		//		npar.SetMetricName(metric)
		params.MetricName = aws.String(metric)
		resp, err := svc.GetMetricStatistics(&params)
		if err != nil {
			return nil, fmt.Errorf("Metric query failed: %s", err.Error())
		}

		m := Metric{
			Label:      *resp.Label,
			Units:      *resp.Datapoints[0].Unit,
			Statistics: "Average",
			Value:      *resp.Datapoints[0].Average,
		}
		mq.Items = append(mq.Items, m)
	}
	sort.Sort(ByLabel(mq.Items))

	return &mq, nil
}

func getMetricDetail(name, timeframe string) (*cloudwatch.GetMetricStatisticsOutput, error) {

	var duration time.Duration
	var period int64

	switch timeframe {
	case "24 hours":
		duration, _ = time.ParseDuration("-24h")
		period = 3600 // 1 hr
	default:
		duration, _ = time.ParseDuration("-4h")
		period = 300 // 5min
	}
	t := time.Now()
	s := t.Add(duration)
	params := cloudwatch.GetMetricStatisticsInput{
		EndTime:    aws.Time(t),
		Namespace:  aws.String(base.Namespace),
		Period:     aws.Int64(period),
		MetricName: aws.String(name),
		StartTime:  aws.Time(s),
		Statistics: []*string{
			aws.String("Average"),
		},
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String(base.DimName),
				Value: aws.String(base.DimValue),
			},
		},
	}
	return svc.GetMetricStatistics(&params)

}

func statQuery() (*EC2MetricsQuery, error) {
	// must be set before calling cloudtap.Getstatistics
	base = &MetricBaseParams{
		DimName:   "InstanceId",
		DimValue:  "xxxxxxxxxxx",
		Namespace: "AWS/EC2",
	}
	// give cloudtap pkg a list of metrics to query
	var metricLabel []string
	for m, _ := range thresh {
		metricLabel = append(metricLabel, m)
	}
	resp, err := getStatistics(metricLabel)
	if err != nil {
		return nil, fmt.Errorf("getStatistics failed: %s", err)
	}
	for i, _ := range resp.Items {
		resp.Items[i].Alert = compareThresh(resp.Items[i])
	}
	// return EC@MetricsQuery object pointer
	return resp, nil
}

// function to compare threshold with query values and return html ready warning

func compareThresh(q Metric) string {
	var minwarn float64 = 0.0
	var maxwarn float64 = 100.0
	var mincrit float64 = 0.0
	var maxcrit float64 = 100.0
	warnings := strings.Split(thresh[q.Label][0], ":")
	if len(warnings) < 2 {
		minwarn = 0
		maxwarn, _ = strconv.ParseFloat(warnings[0], 64)
	} else {
		minwarn, _ = strconv.ParseFloat(warnings[0], 64)
		maxwarn, _ = strconv.ParseFloat(warnings[1], 64)
	}
	criticals := strings.Split(thresh[q.Label][1], ":")
	if len(criticals) < 2 {
		mincrit = 0.0
		maxcrit, _ = strconv.ParseFloat(criticals[0], 64)
	} else {
		mincrit, _ = strconv.ParseFloat(criticals[0], 64)
		maxcrit, _ = strconv.ParseFloat(criticals[1], 64)
	}
	if q.Value > maxcrit || q.Value < mincrit {
		return "danger"
	}
	if q.Value > maxwarn || q.Value < minwarn {
		return "warning"
	}
	return "success"
}

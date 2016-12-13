package main

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"sort"
	"strconv"
	"strings"
	"time"
)

const ID string = "i-029380affdd1af297"

var svc *cloudwatch.CloudWatch
var svc_ec2 *ec2.EC2

type EC2MetricsQuery struct {
	Host      string      `json:"hostname"`
	Namespace string      `json:"namespace"`
	Dims      []Dimension `json:"dimensions"`
	//QNames    []string
	Time  string
	Items []Metric `json:"metrics"`
}
type Dimension struct {
	DimName  string `json:"dim_name"`
	DimValue string `json:"dim_value"`
}

type Metric struct {
	Label      string `json:"metric"`
	Units      string
	Statistics string `json:"statistics"`
	Warning    string `json:"warning"`
	Critical   string `json:"critical"`
	Alert      string
	Value      float64
	Time       float64
}

// sort functions
type ByLabel []Metric
type ByTime []Metric

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

func (mq *EC2MetricsQuery) getStatistics() error {

	mq.Time = time.Now().Format(time.RFC822)
	t := time.Now()
	duration, _ := time.ParseDuration("-48h")
	s := t.Add(duration)
	var dims []*cloudwatch.Dimension
	for i := 0; i < len(mq.Dims); i++ {
		dims = append(dims, &cloudwatch.Dimension{
			Name:  aws.String(mq.Dims[i].DimName),
			Value: aws.String(mq.Dims[i].DimValue),
		})
	}
	params := cloudwatch.GetMetricStatisticsInput{
		EndTime:   aws.Time(t),
		Namespace: aws.String(mq.Namespace),
		Period:    aws.Int64(300),
		//		MetricName: aws.String(metric),
		StartTime:  aws.Time(s),
		Dimensions: dims,
	}
	for i, metric := range mq.Items {
		//		npar.SetMetricName(metric)
		params.MetricName = aws.String(metric.Label)
		params.Statistics = []*string{
			aws.String(metric.Statistics),
		}
		resp, err := svc.GetMetricStatistics(&params)
		if err != nil {
			return fmt.Errorf("Metric query failed: %s", err.Error())
		}
		unit := *resp.Datapoints[len(resp.Datapoints)-1].Unit
		value := *resp.Datapoints[len(resp.Datapoints)-1].Maximum
		if unit == "Bytes" {
			if value > 1048576 {
				value = value / 104857
				unit = "MB"
			} else if value > 1028 {
				value = value / 1028
				unit = "KB"
			}
		}

		mq.Items[i].Units = unit
		mq.Items[i].Value = value
		mq.Items[i].compareThresh()

	}
	sort.Sort(ByLabel(mq.Items))

	return nil
}

func (mq *EC2MetricsQuery) getMetricDetail(stat, name, timeframe string) ([]Metric, error) {

	var duration time.Duration
	var period int64
	var results []Metric

	switch timeframe {
	case "24 hours":
		duration, _ = time.ParseDuration("-24h")
		period = 3600 // 1 hr
	default:
		duration, _ = time.ParseDuration("-4h")
		period = 900 // 5min
	}
	t := time.Now()
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
		Period:     aws.Int64(period),
		MetricName: aws.String(name),
		StartTime:  aws.Time(s),
		Statistics: []*string{
			aws.String(stat),
		},
		Dimensions: dims,
	}
	resp, err := svc.GetMetricStatistics(&params)
	if err != nil {
		return nil, err
	}

	var trans float64 = 1.0
	var tlabel string = ""
	for _, data := range resp.Datapoints {
		// check max values
		if *data.Unit == "Bytes" {
			if *data.Maximum > 1048576.0 {
				trans = 1048576.0
				tlabel = "MB"
			} else if *data.Maximum > 1028.0 {
				trans = 1028.0
				tlabel = "KB"
			}
		}
		m := Metric{
			Label:      *resp.Label,
			Units:      *data.Unit,
			Statistics: "Maximum",
			Value:      *data.Maximum,
			Time:       float64(data.Timestamp.Unix()),
		}
		results = append(results, m)
	}
	sort.Sort(ByTime(results))
	// iterate through metrics and transform for graph
	if trans > 1 {
		for i, _ := range results {
			results[i].Value = results[i].Value / trans
			results[i].Units = tlabel
		}
	}

	return results, nil

}

// function to compare threshold with query values and return html ready warning

func (q *Metric) compareThresh() {
	// adjust for transform
	if q.Units == "MB" {
		q.Value = q.Value * 1048576.0
	}
	if q.Units == "KB" {
		q.Value = q.Value * 1024.0
	}

	var minwarn float64 = 0.0
	var maxwarn float64 = 100.0
	var mincrit float64 = 0.0
	var maxcrit float64 = 100.0
	warnings := strings.Split(q.Warning, ":")
	if len(warnings) < 2 {
		minwarn = 0
		maxwarn, _ = strconv.ParseFloat(warnings[0], 64)
	} else {
		minwarn, _ = strconv.ParseFloat(warnings[0], 64)
		maxwarn, _ = strconv.ParseFloat(warnings[1], 64)
	}
	criticals := strings.Split(q.Critical, ":")
	if len(criticals) < 2 {
		mincrit = 0.0
		maxcrit, _ = strconv.ParseFloat(criticals[0], 64)
	} else {
		mincrit, _ = strconv.ParseFloat(criticals[0], 64)
		maxcrit, _ = strconv.ParseFloat(criticals[1], 64)
	}
	q.Alert = "success"
	if q.Value > maxcrit || q.Value < mincrit {
		q.Alert = "danger"
	} else if q.Value > maxwarn || q.Value < minwarn {
		q.Alert = "warning"
	}

}

func checkInstance() error {
	params := &ec2.DescribeInstancesInput{
		DryRun: aws.Bool(false),
		Filters: []*ec2.Filter{
			{
				Name: aws.String("instance-id"),
				Values: []*string{
					aws.String(ID),
				},
			},
		},
	}
	resp, err := svc_ec2.DescribeInstances(params)

	if err != nil {
		return err
	}

	code := *resp.Reservations[0].Instances[0].State.Code
	if code != 16 {
		es := fmt.Sprintf("Instance %s not running! Code: %d \n", ID, code)
		return errors.New(es)
	}
	return nil
}

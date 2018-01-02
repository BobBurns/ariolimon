package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func load_avg_loop(wg sync.WaitGroup) {

	defer wg.Done()

	dims := []*cloudwatch.Dimension{}
	dim1 := &cloudwatch.Dimension{
		Name:  aws.String("InstanceId"),
		Value: aws.String("i-009614d62f6578510"),
	}
	dims = append(dims, dim1)

	md := new(cloudwatch.MetricDatum)
	md.SetDimensions(dims)

	var a [4]float64
	var b [4]float64
	var c [4]float64
	var d [4]float64
	var e [4]float64
	var la1, la2, la3 float64
	var cpun string

	file, err := os.Open("/proc/stat")
	if err != nil {
		panic(err)
	}

	/* get first average */
	fmt.Fscanf(file, "%s %f %f %f %f", &cpun, &a[0], &a[1], &a[2], &a[3])
	file.Close()

	log.Println(cpun)
	log.Println(a[0], a[1], a[2], a[3])

	/* loop 5 minute averages and publsh */
	count := 0
	for {

		if count == 3 {
			file, err = os.Open("/proc/stat")
			if err != nil {
				panic(err)
			}

			fmt.Fscanf(file, "%s %f %f %f %f", &cpun, &d[0], &d[1], &d[2], &d[3])
			file.Close()

			log.Println(d[0], d[1], d[2], d[3])
			la3 = ((d[0] + d[1] + d[2]) - (a[0] + a[1] + a[2])) / ((d[0] + d[1] + d[2] + d[3]) - (a[0] + a[1] + a[2] + a[3]))

			log.Printf("cpu load avg 15 min: %f\n", la3)
			count = 0
			a[0] = d[0]
			a[1] = d[1]
			a[2] = d[2]
			a[3] = d[3]
			continue
		}

		if count > 0 {

			file, err = os.Open("/proc/stat")
			if err != nil {
				panic(err)
			}

			fmt.Fscanf(file, "%s %f %f %f %f", &cpun, &c[0], &c[1], &c[2], &c[3])
			file.Close()

			log.Println(c[0], c[1], c[2], c[3])
			la2 = ((c[0] + c[1] + c[2]) - (a[0] + a[1] + a[2])) / ((c[0] + c[1] + c[2] + c[3]) - (a[0] + a[1] + a[2] + a[3]))

			log.Printf("cpu load avg 5 min: %f\n", la1)
		}

		e[0] = a[0]
		e[1] = a[1]
		e[2] = a[2]
		e[3] = a[3]
		for i := 0; i < 5; i++ {
			time.Sleep(time.Duration(1 * time.Minute))

			file, err = os.Open("/proc/stat")
			if err != nil {
				panic(err)
			}

			fmt.Fscanf(file, "%s %f %f %f %f", &cpun, &b[0], &b[1], &b[2], &b[3])
			file.Close()

			log.Println(b[0], b[1], b[2], b[3])
			la1 = ((b[0] + b[1] + b[2]) - (e[0] + e[1] + e[2])) / ((b[0] + b[1] + b[2] + b[3]) - (e[0] + e[1] + e[2] + e[3]))

			log.Printf("cpu load avg 1 min: %f\n", la1)

			/* write all metrics every 5 minutes */
			if i == 3 {
				write_cpu_metric(la1, la2, la3, md)
			}
			e[0] = b[0]
			e[1] = b[1]
			e[2] = b[2]
			e[3] = b[3]
		}

		count++

	}

}

func write_cpu_metric(la1, la2, la3 float64, md *cloudwatch.MetricDatum) {
	md.SetMetricName("cpu_load1")
	md.SetTimestamp(time.Now())
	md.SetUnit("Percent")
	md.SetValue(la1 * 100)

	err := md.Validate()
	if err != nil {
		panic(err)
	}
	pmdin := &cloudwatch.PutMetricDataInput{
		MetricData: []*cloudwatch.MetricDatum{md},
		Namespace:  aws.String("CWBobby"),
	}

	log.Println(pmdin)
	out, err := svc.PutMetricData(pmdin)
	if err != nil {
		panic(err)
	}
	log.Println("response: ", out)

	md.SetMetricName("cpu_load5")
	md.SetValue(la2 * 100)
	err = md.Validate()
	if err != nil {
		panic(err)
	}
	pmdin = &cloudwatch.PutMetricDataInput{
		MetricData: []*cloudwatch.MetricDatum{md},
		Namespace:  aws.String("CWBobby"),
	}

	log.Println(pmdin)
	out, err = svc.PutMetricData(pmdin)
	if err != nil {
		panic(err)
	}
	log.Println("response: ", out)

	md.SetMetricName("cpu_load15")
	md.SetValue(la3 * 100)
	err = md.Validate()
	if err != nil {
		panic(err)
	}
	pmdin = &cloudwatch.PutMetricDataInput{
		MetricData: []*cloudwatch.MetricDatum{md},
		Namespace:  aws.String("CWBobby"),
	}

	log.Println(pmdin)
	out, err = svc.PutMetricData(pmdin)
	if err != nil {
		panic(err)
	}
	log.Println("response: ", out)
}

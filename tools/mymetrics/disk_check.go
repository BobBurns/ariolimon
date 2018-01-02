package main

import (
	"log"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

/* push metrics every 3 minutes */
func get_disk_loop(wg sync.WaitGroup) {

	defer wg.Done()

	dims := []*cloudwatch.Dimension{}
	dim1 := &cloudwatch.Dimension{
		Name:  aws.String("InstanceId"),
		Value: aws.String("i-009614d62f6578510"),
	}
	dims = append(dims, dim1)

	md := new(cloudwatch.MetricDatum)
	md.SetDimensions(dims)

	var status syscall.Statfs_t
	var fperc float64

	for {
		/* check "/" for now */
		err := syscall.Statfs("/", &status)
		if err != nil {
			panic(err)
		}

		bav := float64(status.Bavail * uint64(status.Bsize))
		bfr := float64(status.Blocks * uint64(status.Frsize))

		fperc = bav / bfr

		md.SetMetricName("disk_free:/")
		md.SetTimestamp(time.Now())
		md.SetUnit("Percent")
		md.SetValue(fperc * 100)

		err = md.Validate()
		if err != nil {
			panic(err)
		}

		/* push disk free metric */
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

		md.SetMetricName("disk_avail:/")
		md.SetTimestamp(time.Now())
		md.SetUnit("Bytes")
		md.SetValue(bav)

		err = md.Validate()
		if err != nil {
			panic(err)
		}

		/* push disk available metric */
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

		/* sleep and loop */
		time.Sleep(time.Duration(3 * time.Minute))
	}
}

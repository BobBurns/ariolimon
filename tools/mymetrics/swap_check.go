package main

import (
	"bufio"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func check_swap_loop(wg sync.WaitGroup) {
	defer wg.Done()

	dims := []*cloudwatch.Dimension{}
	dim1 := &cloudwatch.Dimension{
		Name:  aws.String("InstanceId"),
		Value: aws.String("i-009614d62f6578510"),
	}
	dims = append(dims, dim1)

	md := new(cloudwatch.MetricDatum)
	md.SetDimensions(dims)

	var swapFree float64 = 0.0
	var swapTotal float64 = 0.0

	sfRe := regexp.MustCompile("SwapFree\\:")
	stRe := regexp.MustCompile("SwapTotal\\:")

	for {
		file, err := os.Open("/proc/meminfo")
		if err != nil {
			panic(err)
		}

		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			t := scanner.Text()
			if sfRe.MatchString(t) {
				ts := strings.Fields(t)
				sf, _ := strconv.Atoi(ts[1])
				swapFree = float64(sf)
			}

			if stRe.MatchString(t) {
				ts := strings.Fields(t)
				st, _ := strconv.Atoi(ts[1])
				swapTotal = float64(st)
			}
		}
		file.Close()

		md.SetMetricName("swap_free")
		md.SetTimestamp(time.Now())
		md.SetUnit("Bytes")
		md.SetValue(swapFree)

		err = md.Validate()
		if err != nil {
			panic(err)
		}

		/* push swap free */
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

		md.SetMetricName("swap_total")
		md.SetTimestamp(time.Now())
		md.SetUnit("Bytes")
		md.SetValue(swapTotal)

		err = md.Validate()
		if err != nil {
			panic(err)
		}

		/* push swap total */
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

		time.Sleep(time.Duration(3 * time.Minute))
	}

}

/* example of pushing custom metrics to AWS CloudWatch */
/* use on ec2 instance */

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

var svc *cloudwatch.CloudWatch

func main() {

	/* new session */
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)
	if err != nil {
		panic(err)
	}

	svc = cloudwatch.New(sess)

	/* log output */
	f, err := os.OpenFile("cwtest.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	defer f.Close()
	log.SetOutput(f)

	var wg sync.WaitGroup

	wg.Add(1)
	go get_disk_loop(wg)

	wg.Add(1)
	go load_avg_loop(wg)

	wg.Add(1)
	go check_swap_loop(wg)

	/* shouldn't
	 * reach
	 * until
	 * interupt
	 * */
	wg.Wait()

}

func get_disk_loop(wg sync.WaitGroup) {

	defer wg.Done()
	/* new
	* session
	* */

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
		/* check
		 * only
		 * "/"
		 * for
		 * now
		 * */
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
		/*publish
		 * disk
		 * free
		 * */
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
		/*publish
		 * disk
		 * availible
		 * bytes
		 * */
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

		/* sleep
		 * and
		 * loop
		 * */
		time.Sleep(time.Duration(3 * time.Minute))
	}
}

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

	/* get
	 * first
	 * average
	 * */
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
		/* every
		 * 5
		 * minutes
		 * count
		 * increases
		 * by
		 * 1
		 * */
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

			/* push
			 * metric
			 * every
			 * 5
			 * minutes
			 * */
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

func check_swap_loop(wg sync.WaitGroup) {
	defer wg.Done()

	/* new
	 * session
	 * */
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)
	if err != nil {
		panic(err)
	}

	svc := cloudwatch.New(sess)

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
		/*publish
		 * swap
		 * free
		 * */
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
		/*publish
		 * swap
		 * total
		 * */
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

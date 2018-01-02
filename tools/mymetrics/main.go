package main

import (
	"log"
	"os"
	"sync"

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

	/* loop metric checks on seperate threads */
	var wg sync.WaitGroup

	wg.Add(1)
	go get_disk_loop(wg)
	wg.Add(1)
	go load_avg_loop(wg)
	wg.Add(1)
	go check_swap_loop(wg)

	/* shouldn't reach until interupt */
	wg.Wait()

}

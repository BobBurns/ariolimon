##**Ariolimon**
A go web application to query Amazon web services Metrics and visually warn if configured thresholds are exceeded.

###**Please note this program is under active development**###
Currently you can query only one instance

###**Installation**
To use, you need an [AWS](https://aws.amazon.com/) EC2 instance and the [AWS CLI](http://docs.aws.amazon.com/cli/latest/userguide/installing.html) with your [credentials](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html).

If you don't already have it, you can download and install Go from [here](https://golang.org/dl/).


Create an ariolimon directory inside your $GOPATH.

cd into ariolimon and clone this repository.

Change the thresholds.conf file to suit your needs.  The syntax is
```
AwsMetric	<lo warning value>:<hi warning value> <lo critical value>:<hi critical value>
```
so the following line in thresholds.conf
```
CPUUtilization			0:5	0:50
```
means warn if CPUUtilization is higher than 5 and warn critical when higher than 50

In logic_model.go replace xxx in ```const ID string = "xxxxxxxxxxxxxxxxxx"``` with your instance id.

Run ```go build ariolimon``` and execute ```../ariloimon```

open a web browser and navigate to localhost:8082.

Feel free to contact me for troubleshooting reburns@protonmail.com

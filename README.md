##**Ariolimon**
A go web application to query Amazon web services Metrics and visually warn if configured thresholds are exceeded.

###**Please note this program is under active development**###

###**Installation**
To use, you need an [AWS](https://aws.amazon.com/) EC2 instance and the [AWS CLI](http://docs.aws.amazon.com/cli/latest/userguide/installing.html) with your [credentials](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html).

If you don't already have it, you can download and install Go from [here](https://golang.org/dl/).

Clone this repo inside your $GOPATH.

Change the thresh2.json file to suit your needs.

Run ```go build ``` and execute ```./adev```

Open a web browser and navigate to localhost:8082

Feel free to contact me for troubleshooting reburns@protonmail.com

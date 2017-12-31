# **Ariolimon** #

A go web application to query Amazon web services Metrics and visually warn if configured thresholds are exceeded.


## **Installation** ##

To use, you need an [AWS](https://aws.amazon.com/) EC2 instance and the [AWS CLI](http://docs.aws.amazon.com/cli/latest/userguide/installing.html) with your [credentials](http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html).

Ensure there is ~/.aws/config and ~/.aws/credentials set correctly

If you don't already have it, you can download and install Go from [here](https://golang.org/dl/).

**Install dependencies**

```
go get
```

Clone this repo inside your $GOPATH.

Change the thresh.json file to suit your needs.

Add a configdb.json file in the main directory that has database authentication info

```
{
  "Host"	: "127.0.0.1",
  "User"	: "user",
  "Pass"	: "password",
  "Db"		: "aws_metric_store"
}
```

Note if you are running go 1.8 comment out line 41 in main.go

Start database server `mongod --auth`

`go build new_user.go` in new\_user directory and run `./new_user -u user <-p password>` 

Note if you are running < go 1.8 comment out line 41 in main.go

Run `go build ` and execute `./ariolimon`

Open a web browser and navigate to localhost:8082

Feel free to contact me for troubleshooting reburns@protonmail.com

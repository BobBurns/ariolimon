### *mymetrics*

Here are a few examples of creating custom metrics to publish on AWS CloudWatch.

Build and Run on your ec2 instance with proper AWS credentials.  

`go build`

`./mymetrics`

If you would like to monitor with Ariolimon add something like this to thresh.json

```json
{"name": "apm1_disk_free",
 "hostname": "apm-prod-1",
 "namespace": "CWBobby",
 "dimensions": [
	{"dim_name": "InstanceId",
	 "dim_value": "i-009614d62f6578510"}
 ],
 "metric": "disk_free:/",
 "statistics": "Minimum",
 "warning": "10:",
"critical": "5:"},
```

Note that you cannot use some characters like / in the name parameter

Happy Monitoring!


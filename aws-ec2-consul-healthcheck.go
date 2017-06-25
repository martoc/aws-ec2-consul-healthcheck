package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"time"
	"net/http"
	"io/ioutil"
	"encoding/json"
)

var canAwsSetInstanceHealth bool
var tr = &http.Transport{
	MaxIdleConns:       10,
	IdleConnTimeout:    30 * time.Second,
	DisableCompression: true,
}
var httpClient = &http.Client{Transport: tr}

func main() {
	fmt.Println("Initialising healthcheck...")
	serviceName := flag.String("service-name", "", "Path of consul service file")
	graceInterval := flag.Duration("grace-interval", 0, "Grace interval in seconds")
	interval := flag.Duration("interval", 0, "Health check timeout in seconds")
	unhealthyThreshold := flag.Int("unhealthy-threshold", 0, "Number of failed health checks before the machine is assumed unhealthy")
	flag.BoolVar(&canAwsSetInstanceHealth, "aws-set-instance-health", true, "Indicates aws unhealthy call should be make")
	flag.Parse()
	fmt.Printf("Service Name: %s, grace interval: %s, interval: %s, unhealthy threshold: %d \n",
		*serviceName, *graceInterval, *interval, *unhealthyThreshold)
	time.Sleep(*graceInterval)
	counter := 0
	for {
		time.Sleep(*interval)
		if isHealthy(*serviceName) {
			counter = 0
			fmt.Println("Service Healthy no actions ot take")
			setInstanceHealth("Healthy")
		} else {
			if counter >= *unhealthyThreshold {
				fmt.Println("Service Unhealthy unhealthyThreshold reached, taking actions")
				setInstanceHealth("Unhealthy")
			} else {
				fmt.Println("Service Unhealthy unhealthyThreshold has not been reached")
				counter++
			}
		}
	}
}

func isHealthy(serviceName string) bool {
	selfContent, error := getContent("http://localhost:8500/v1/agent/checks")
	if error != nil {
		return false
	}
	var jsonObject interface{}
	json.Unmarshal(selfContent, &jsonObject)
	services := jsonObject.(map[string]interface{})
	agentServiceName := "service:" + serviceName
	if services[agentServiceName] != nil {
		service := services[agentServiceName].(map[string]interface{})
		fmt.Printf("Service status: %s\n", service["Status"])
		return service["Status"] == "passing"
	}
	return true
}


func getContent(path string) (body []byte, err error) {
	resp, error := httpClient.Get(path)
	if error != nil {
		return nil, error
	}
	body, error = ioutil.ReadAll(resp.Body)
	if error != nil {
		return body, error
	}
	defer resp.Body.Close()
	return body, nil
}

func setInstanceHealth(health string) {
	if canAwsSetInstanceHealth {
		awsSetInstanceHealth(health)
	}
}

func awsSetInstanceHealth(health string) {
	session := session.Must(session.NewSession())
	ec2metadataService := ec2metadata.New(session)
	autoscalingService := autoscaling.New(session)
	region, _ := ec2metadataService.Region()
	instanceId, _ := ec2metadataService.GetMetadata("instance-id")
	shouldRespectGracePeriod := true
	fmt.Printf("Region: %s and instanceId: %s\n", region, instanceId)
	request := autoscaling.SetInstanceHealthInput{HealthStatus: &health, InstanceId: &instanceId,
		ShouldRespectGracePeriod: &shouldRespectGracePeriod}
	autoscalingService.SetInstanceHealth(&request)
}

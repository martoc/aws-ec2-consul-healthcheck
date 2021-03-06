package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"time"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"strings"
)

var canAwsSetInstanceHealth bool
var tr = &http.Transport{
	MaxIdleConns:       10,
	IdleConnTimeout:    30 * time.Second,
	DisableCompression: true,
}
var httpClient = &http.Client{Transport: tr}
var awsRegion string
var awsInstanceId string

func main() {
	fmt.Println("Initialising healthcheck...")
	servicePath := flag.String("service-path", "", "Path of consul service file")
	graceInterval := flag.Duration("grace-interval", 0, "Grace interval in seconds")
	interval := flag.Duration("interval", 0, "Health check timeout in seconds")
	unhealthyThreshold := flag.Int("unhealthy-threshold", 0, "Number of failed health checks before the machine is assumed unhealthy")
	flag.BoolVar(&canAwsSetInstanceHealth, "aws-set-instance-health", true, "Indicates aws unhealthy call should be make")
	flag.Parse()
	fmt.Printf("Service Path: %s, grace interval: %s, interval: %s, unhealthy threshold: %d \n",
		*servicePath, *graceInterval, *interval, *unhealthyThreshold)
	time.Sleep(*graceInterval)
	if canAwsSetInstanceHealth {
		instanceId, _ := GetContent("http://169.254.169.254/latest/meta-data/instance-id")
		region, _ := GetContent("http://169.254.169.254/latest/meta-data/placement/availability-zone")
		awsInstanceId = string(instanceId)
		regionString := string(region)
		awsRegion = regionString[:len(regionString) - 1]
		fmt.Printf("Region: %s and instanceId: %s\n", awsRegion, awsInstanceId)
	}
	counter := 0
	serviceNames := GetServiceNames(*servicePath)
	for {
		time.Sleep(*interval)
		if IsHealthy(serviceNames) {
			counter = 0
			fmt.Println("Service Healthy no actions to take")
			SetInstanceHealth("Healthy")
		} else {
			if counter >= *unhealthyThreshold {
				fmt.Println("Service Unhealthy unhealthyThreshold reached, taking actions")
				SetInstanceHealth("Unhealthy")
			} else {
				fmt.Println("Service Unhealthy unhealthyThreshold has not been reached")
				counter++
			}
		}
	}
}

func IsHealthy(serviceNames []string) bool {
	selfContent, error := GetContent("http://localhost:8500/v1/agent/checks")
	if error != nil {
		return false
	}
	var jsonObject interface{}
	json.Unmarshal(selfContent, &jsonObject)
	services := jsonObject.(map[string]interface{})
	generalHealth := true
	for _, serviceName := range serviceNames {
		agentServiceName := "service:" + serviceName
		if services[agentServiceName] != nil {
			service := services[agentServiceName].(map[string]interface{})
			fmt.Printf("Service %s: %s\n", serviceName, service["Status"])
			generalHealth = generalHealth && service["Status"] == "passing"
		}
	}
	return generalHealth
}

func GetServiceNames(servicePath string) (serviceNames []string) {
	files, _ := ioutil.ReadDir(servicePath)
	for _, f := range files {
		filename := f.Name()
		if !strings.HasPrefix(filename, ".") && filename != "consul.json" {
			content, _ := ioutil.ReadFile(servicePath + "/" + filename)
			var jsonObject interface{}
			json.Unmarshal(content, &jsonObject)
			services := jsonObject.(map[string]interface{})
			service := services["service"].(map[string]interface{})
			fmt.Printf("Service Name: %s\n", service["name"])
			serviceNames = append(serviceNames, service["name"].(string))
		}
	}
	return serviceNames
}


func GetContent(path string) (body []byte, err error) {
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

func SetInstanceHealth(health string) {
	if canAwsSetInstanceHealth {
		AwsSetInstanceHealth(health)
	}
}

func AwsSetInstanceHealth(health string) {
	session := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(awsRegion),
	}))
	shouldRespectGracePeriod := true
	autoscalingService := autoscaling.New(session)
	request := autoscaling.SetInstanceHealthInput{HealthStatus: &health, InstanceId: &awsInstanceId,
		ShouldRespectGracePeriod: &shouldRespectGracePeriod}
	respose, error := autoscalingService.SetInstanceHealth(&request)
	if error != nil {
		fmt.Printf("Error updating health: %v\n", error)
	} else {
		fmt.Printf("Result of updating health: %v\n", respose)
	}
}

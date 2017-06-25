deps:
	go get github.com/aws/aws-sdk-go

build: deps
	go build main.go

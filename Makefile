deps:
	go get github.com/aws/aws-sdk-go

build: deps
	rm -rf target/*
	mkdir -p target
	go build -o target/aws-ec2-consul-healthcheck aws-ec2-consul-healthcheck.go
	cd target && tar -cvf aws-ec2-consul-healthcheck_0.0.1_linux_amd64.tar.gz .

build-centos7:
	mkdir -p target
	docker build --no-cache=true -f ./sandbox/Dockerfile.centos7 -t "aws-ec2-consul-healcheck-centos7" .
	docker run -i -v ${PWD}/target:/go/aws-ec2-consul-healthcheck/target -t "aws-ec2-consul-healcheck-centos7"

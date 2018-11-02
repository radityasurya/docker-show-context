install:
	GO111MODULE=on go install

build:
	go build -o bin/docker-show-context

run:
	go run main.go

.PHONY: build get install run 

BINARY=aws-login

.PHONY: install build

install:
	go install ./cmd/aws-login

build:
	mkdir -p bin
	go build -o bin/$(BINARY) ./cmd/aws-login

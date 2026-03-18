BINARY=aws-login

.PHONY: install build

install:
	go install .

build:
	mkdir -p bin
	go build -o bin/$(BINARY) .

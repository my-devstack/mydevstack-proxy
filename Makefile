SHELL:=/bin/bash

PROJECT_PATH := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
TEST_OUTPUT_PATH := $(PROJECT_PATH)/.output

FILE=./.env
ifneq ("$(wildcard $(FILE))","")
	include $(FILE)
	export $(shell sed 's/=.*//' $(FILE))
endif

.PHONY: run
run:
	go mod tidy && go run cmd/server/main.go
	

.PHONY: mockery
mockery:
	mockery && go mod tidy

.PHONY: unit
unit:
	go mod tidy
	go test $(shell go list ./internal/... | grep -v /mocks) -race -coverprofile .testCoverage.txt -v 2>&1
.PHONY: unit-coverage
unit-coverage: unit ## Runs unit tests and generates a html coverage report
	go tool cover -html=.testCoverage.txt -o unit.html
VERSION=$(shell git describe)

.PHONY: build arm

build:
	go build -ldflags "-X main.version=$(VERSION)"

arm:
	GOOS=linux GOARCH=arm go build -ldflags "-X main.version=$(VERSION)"

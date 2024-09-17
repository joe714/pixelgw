# Makefile to run inside the build container
.PHONY: prep generate build

build: generate
	go build -o bin/ cmd/pixelgw/pixelgw.go

generate:
	go generate internal/api/pixelgw.go

all: build


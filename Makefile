
all: fmt build
run:fmt build
	./i2pjump

build:
	go build

fmt:
	gofmt -w -s *.go */*.go


all: fmt build
run:fmt build
	./i2pjump

build:
	go build

fmt:
	gofmt -w -s *.go */*.go
	
	
	
GO111MODULE=auto

JAVA_HOME=/usr/lib/jvm/java-8-openjdk-amd64/

jar: copy
	gojava -v -o i2pjump.jar build ./lib

uncopy:
	rm -rfv "$(HOME)/go/pkg/mod/github.com/sridharv/gojava"

copy: uncopy
	cp -rv "$(HOME)/go/src/github.com/sridharv/gojava" "$(HOME)/go/pkg/mod/github.com/sridharv/gojava"

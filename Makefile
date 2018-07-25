GOCMD=go
GOCLEAN=$(GOCMD) clean
GOBUILD=$(GOCMD) build
GOFMT=$(GOCMD) fmt
GOTEST=$(GOCMD) test
BINARY_NAME=nekoq-bootstrap

all: test build
build:
	$(GOFMT) ./...
	$(GOBUILD) -v
test:
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
run:
	$(GOBUILD) -v
	./$(BINARY_NAME)
.PHONY: all

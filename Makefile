GOCMD=go
GOCLEAN=$(GOCMD) clean
GOBUILD=$(GOCMD) build
GOFMT=$(GOCMD) fmt
GOTEST=$(GOCMD) test
BINARY_NAME=nekoq-bootstrap

all: clean test build
build:
	$(GOFMT) ./...
	$(GOBUILD) -v
test:
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
run:
	$(GOFMT) ./...
	$(GOBUILD) -v
	./$(BINARY_NAME) -port=10053

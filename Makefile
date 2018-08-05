GOCMD=go
GOCLEAN=$(GOCMD) clean
GOBUILD=$(GOCMD) build
GOFMT=$(GOCMD) fmt
GOTEST=$(GOCMD) test
BINARY_NAME=nekoq-bootstrap

all: clean test build
build:
	$(GOFMT) ./...
	$(GOBUILD) -v -o ./$(BINARY_NAME) ./cmd
test:
	$(GOTEST) -v ./...
clean:
	rm -rf ./$(BINARY_NAME)
run:
	$(GOFMT) ./...
	$(GOBUILD) -v -o ./$(BINARY_NAME) ./cmd
	./$(BINARY_NAME) -port=10053

BINARY_NAME=photoepics

all: test build
build:
	go build -o $(BINARY_NAME) -v

test:
	go vet ./...
	go test ./...

clean:
	go clean
	rm -f $(BINARY_NAME)

run:
	go build -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)

deps:
	go get

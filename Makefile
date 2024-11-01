
clean:
	go clean
	go mod tidy -v
	find . -type f -name .DS_Store -delete

fmt :
	@echo "Formatting your Go programs with gofmt..."
	@gofmt -l -w ./

lint :
	@echo "Using golangci-lint to detect your code quality ..."
	@golangci-lint run

test :
	@echo "Testing your code, please wait ..."
	@go test -race -cover -coverprofile=coverage.txt -covermode=atomic ./...

check: fmt lint test
	@echo "Prepare to release the code, are performing code checking, please wait ..."
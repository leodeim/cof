.PHONY: test test-race bench cover cover-html vet lint clean

test:
	go test -count=1 ./...

test-race:
	go test -race -count=1 -timeout=2m ./...

bench:
	go test -bench=. -benchmem -count=1 -timeout=2m ./...

cover:
	go test -race -count=1 -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

cover-html: cover
	go tool cover -html=coverage.out -o coverage.html

vet:
	go vet ./...

lint: vet
	@which golangci-lint > /dev/null 2>&1 || { echo "golangci-lint not installed: https://golangci-lint.run/welcome/install/"; exit 1; }
	golangci-lint run ./...

clean:
	rm -f coverage.out coverage.html

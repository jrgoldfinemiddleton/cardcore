.PHONY: test fmt vet lint build doc check

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

lint:
	go tool golangci-lint run

build:
	go build ./...

doc:
	go tool pkgsite -open .

check: fmt vet lint test

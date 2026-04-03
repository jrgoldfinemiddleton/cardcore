.PHONY: test fmt vet lint build doc check

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

lint:
	golangci-lint run

build:
	go build ./...

doc:
	pkgsite -open .

check: fmt vet lint test

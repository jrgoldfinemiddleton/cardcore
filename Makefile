.PHONY: test fmt vet lint lint-extra build doc check create-labels apply-labels

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

lint:
	go tool golangci-lint run

lint-extra:
	go tool golangci-lint run --config .golangci-extra.yml

build:
	go build ./...

doc:
	go tool pkgsite -open .

check: fmt vet lint test

create-labels:
	./scripts/sync-labels.sh

apply-labels:
	@if [ -z "$(PR)" ]; then echo "usage: make apply-labels PR=<pr-number>" >&2; exit 1; fi
	./scripts/apply-labels.sh $(PR)

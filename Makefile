GO_VERSION=1.22.1
GO_LDFLAGS=-extldflags '-lnvidia-ml'
MODULE="github.com/klueska/nvkind"

all: vendor fmt build

TARGETS := vendor fmt lint vet test build run
.PHONY: $(TARGETS)

fmt:
	go list -f '{{.Dir}}' $(MODULE)/... \
		| xargs gofmt -s -l -w

lint:
	golangci-lint run ./...

vet:
	go vet $(MODULE)/...

vendor:
	go mod tidy
	go mod vendor
	go mod verify

test:
	go test -ldflags "$(GO_LDFLAGS)" $(MODULE)/...

build:
	go build -ldflags "$(GO_LDFLAGS)" $(MODULE)/cmd/...

run:
	go run -ldflags "$(GO_LDFLAGS)" $(MODULE)/...

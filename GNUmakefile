default: build

HOSTNAME=registry.terraform.io
NAMESPACE=rivestack
NAME=rivestack
BINARY=terraform-provider-${NAME}
VERSION=0.1.0
OS_ARCH=$(shell go env GOOS)_$(shell go env GOARCH)

build:
	go build -o ${BINARY}

install: build
	mkdir -p ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}
	mv ${BINARY} ~/.terraform.d/plugins/${HOSTNAME}/${NAMESPACE}/${NAME}/${VERSION}/${OS_ARCH}

lint:
	golangci-lint run ./...

test:
	go test ./... -v -count=1 -parallel=4

testacc:
	TF_ACC=1 go test ./... -v -count=1 -parallel=1 -timeout 120m

generate:
	go generate ./...

fmt:
	gofmt -s -w .
	terraform fmt -recursive examples/

clean:
	rm -f ${BINARY}

.PHONY: build install lint test testacc generate fmt clean

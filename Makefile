PROG=pandocserver

.DEFAULT_GOAL := build

.PHONY: all
all: update lint build

.PHONY: docker-update
docker-update:
	docker pull golang:latest
	docker pull pandoc/extra:latest
	docker build --tag ${PROG}:dev .

.PHONY: docker-run
docker-run: docker-update
	docker run --init --rm -p 8000:8000 ${PROG}:dev -host 0.0.0.0:8000 -debug

.PHONY: lint
lint:
	"$$(go env GOPATH)/bin/golangci-lint" run ./...
	go mod tidy

.PHONY: lint-update
lint-update:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin
	$$(go env GOPATH)/bin/golangci-lint --version

.PHONY: update
update:
	go get -u
	go mod tidy -v

.PHONY: build
build:
	go fmt ./...
	go vet ./...
	go build -o ${PROG}

.PHONY: test
test:
	go test -race -cover ./...

.PHONY: run
run: build
	 ./${PROG} -host 0.0.0.0:8000 -debug

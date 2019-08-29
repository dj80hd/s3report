COMMIT_SHA=$(shell git rev-parse HEAD)

default: test build cover

.PHONY: dep test cover build linux docker publish clean 

test:
	go vet `go list ./...`
	for pkg in `go list ./...`; do \
		golint -set_exit_status $$pkg || exit 1; \
	done
	go test -timeout 1s ./...

race:
	go test -race -timeout 5s ./...

cover:
	goverage -v -coverprofile=cov.out ./...

build:
	go fmt ./...
	go build 

linux:
	GOOS=linux go build main.go
#	for pkg in `go list ./cmd/...`; do \
#		GOOS=linux go build -ldflags="-X main.Version=$(COMMIT_SHA)" $$pkg || exit 1; \
#	done

clean:
	rm -f s3report

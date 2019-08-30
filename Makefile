default: build

test: build
	go vet `go list ./...`
	for pkg in `go list ./...`; do \
		golint -set_exit_status $$pkg || exit 1; \
	done
	go test -timeout 1s ./...

cover: test
	goverage -v -coverprofile=cov.out ./...

build:
	go fmt ./...
	go build -o bin/s3report

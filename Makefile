default: build

test: build
	go test -cover -timeout 1s ./...

build:
	go fmt ./...
	go vet `go list ./...`
	golint ./...
	go build -o bin/s3report

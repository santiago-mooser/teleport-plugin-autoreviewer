.PHONY: test pre-test-go test-go pre-commit pre-doc-go generate generate-go generate-doc generate-event build run-local view-spec build-listener build-connector run-local-listener run-local-connector
.DEFAULT_GOAL=test

SERVICE_NAME=teleport-plugin-request-autoreviewer

# build

pre-commit:
	go mod tidy
	go vet ./...
	go fmt ./...

build:
	env GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -o $(SERVICE_NAME)

generate-go:
	go generate -mod=mod ./...

all: clean deps-dev lint build tests
clean:
	rm -rf bin/*

deps-dev:
	go install github.com/kisielk/errcheck@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/tools/cmd/goimports@latest

lint: deps-dev
	go mod verify
	go vet $$(go list ./... | grep -v /vendor/)
	goimports -w -local gitlab.com/supriya-premkumar/mempoor $$(go list -f {{.Dir}} ./... | grep -v /vendor/)
	@echo "\nError Checking:"
	errcheck --exclude .errcheck ./...
	@echo "\nStatic Checking:"
	staticcheck ./...

build: clean lint
	GOOS=linux  GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o ./bin/mempoor-linux  ./cmd/mempoor
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o ./bin/mempoor-darwin ./cmd/mempoor

.PHONY: tests e2e
tests:
	go test -race ./...

e2e:
	@echo "Running mempoor end to end test..."
	./scripts/e2e.sh
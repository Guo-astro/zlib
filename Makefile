.PHONY: build check fmt hooks tidy vet

build:
	go build github.com/heartleo/zlib/cmd/zlib

# Mandatory before every commit; .githooks/pre-commit runs it.
check:
	sh scripts/check.sh

fmt:
	go fmt ./...

# Run once per clone.
hooks:
	git config core.hooksPath .githooks

tidy:
	go mod tidy

vet:
	go vet ./...

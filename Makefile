.PHONY: setup fmt lint test build clean

## setup: configure git hooks path to scripts/hooks
setup:
	git config core.hooksPath scripts/hooks
	@echo "Git hooks configured. Pre-commit hook is now active."

## fmt: format all Go source files
fmt:
	go fmt ./...

## lint: run go vet on all packages
lint:
	go vet ./...

## test: run all tests
test:
	go test ./...

## build: build the logos binary
build:
	go build -o logos .

## clean: remove the built binary
clean:
	rm -f logos

## help: show this help message
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' Makefile | sed 's/^## /  /'

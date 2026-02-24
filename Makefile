.PHONY: setup fmt lint test build install clean

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

## install: build and install the logos binary to ~/bin
install: build
	cp logos ~/bin/logos

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

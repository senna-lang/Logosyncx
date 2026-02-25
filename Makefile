.PHONY: setup fmt lint test build install clean snapshot release-dry-run release help

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

## build: build the logos binary (dev build — version will show as "dev")
build:
	go build -o logos .

## install: build and install the logos binary to ~/bin
install: build
	cp logos ~/bin/logos

## clean: remove the built binary and GoReleaser output
clean:
	rm -f logos
	rm -rf dist/

## snapshot: build a local snapshot for all platforms using GoReleaser (no publish, no git tag required)
snapshot:
	goreleaser build --snapshot --clean

## release-dry-run: full release dry run — builds all platforms and creates archives but does not publish
release-dry-run:
	goreleaser release --snapshot --clean

## release: tag HEAD and push to trigger the GitHub Actions release pipeline
##          Runs fmt, lint, and test first to catch issues before tagging.
release: fmt lint test
	@current=$$(git describe --tags --abbrev=0 2>/dev/null || echo "none"); \
	echo "Current version tag: $$current"; \
	read -p "New version tag (e.g. v0.2.0): " tag; \
	echo "Tagging $$tag ..."; \
	git tag "$$tag" && git push origin "$$tag"

## help: show this help message
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' Makefile | sed 's/^## /  /'

# Makefile — build/test/lint tasks for memegen.
# Common entry points: `make` (build), `make test`, `make run IMG=foo.png`.

BINARY := memegen
PKG    := ./...
CMD    := ./cmd/memegen

.DEFAULT_GOAL := build

.PHONY: build test vet fmt tidy run clean install ci

build: ## compile the binary
	go build -o $(BINARY) $(CMD)

test: ## run all tests
	go test $(PKG)

vet: ## static analysis
	go vet $(PKG)

fmt: ## format sources
	gofmt -w .

tidy: ## sync go.mod/go.sum
	go mod tidy

run: build ## run editor: make run [IMG=image.png] [OUT=out.png] (no IMG opens browser)
	./$(BINARY) $(if $(OUT),-o $(OUT),) $(IMG)

install: ## install to GOBIN
	go install $(CMD)

clean: ## remove build artifacts
	rm -f $(BINARY)

ci: tidy fmt vet test build ## full local check

help: ## list targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

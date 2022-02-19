PROJECT_NAME := "gf-sframe"
PKG := "github.com/greatfocus/$(PROJECT_NAME)"
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)
 
.PHONY: all dep lint vet test test-coverage build clean
 
all: build

dep: ## Get the dependencies
	@go clean -cache -modcache -i -r
	@go mod download

lint: ## Lint Golang files
	@golint -set_exit_status

vet: ## Run go vet
	@go vet

test: ## Run unittests
	@go test -v -cover -short

test-coverage: ## Run tests with coverage
	@go test -short -coverprofile cover.out -covermode=atomic
	@cat cover.out >> coverage.txt

build: dep ## Build the binary file
	@go build -i -o build/gf-sframe $(PKG)
 
clean: ## Remove previous build
	@rm -f $(PROJECT_NAME)/build

update-pkg-cache:
    GOPROXY=https://proxy.golang.org GO111MODULE=on \
    go get github.com/greatfocus/$(PROJECT_NAME)@v$(VERSION)
 
help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
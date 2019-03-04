# Expands to list this project's go packages, excluding the vendor folder
SHELL = bash
PACKAGES = $$(go list ./... | grep -v /vendor/)
BUILD_FLAGS =

all: fmt build lint test

build:
	go build $(BUILD_FLAGS) $(PACKAGES)

builddir:
	@if [ ! -d build ]; then mkdir build; fi

lint:
	golint -set_exit_status $(PACKAGES)

clean:
	rm -rf build/*

fmt:
	go fmt $(PACKAGES)

test:
	go test $(BUILD_FLAGS) $(PACKAGES)

testreport: builddir
	# runs go test and generate coverage report
	go test $(BUILD_FLAGS) -covermode=count -coverprofile=build/coverage.out $(PACKAGES)
	go tool cover -html=build/coverage.out -o build/coverage.html

bench:
	go test -bench .

vendor.update:
	dep ensure --update

vendor.ensure:
	dep ensure

### TOOLS

tools:
	go get -u github.com/golang/dep/cmd/dep
	go get -u golang.org/x/tools/cmd/cover
	go get -u golang.org/x/lint/golint

.PHONY: all build builddir lint clean fmt test testreport vendor.update vendor.ensure tools


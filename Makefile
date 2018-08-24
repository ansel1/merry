# Expands to list this project's go packages, excluding the vendor folder
SHELL = bash
PACKAGES = $$(go list ./... | grep -v /vendor/)
BUILD_FLAGS =

all: fmt build vet lint test

build:
	go build $(BUILD_FLAGS) $(PACKAGES)

builddir:
	@if [ ! -d build ]; then mkdir build; fi

vet:
	go vet $(PACKAGES)

lint:
	golint -set_exit_status $(PACKAGES)

clean:
	rm -rf build/*

fmt:
	go fmt $(PACKAGES)

test:
	go test $(BUILD_FLAGS) $(PACKAGES)

testreport: builddir
	# runs go test in each package one at a time, generating coverage profiling
    # finally generates a combined junit test report and a test coverage report
    # note: running coverage messes up line numbers in error stacktraces
	go test $(BUILD_FLAGS) -v -covermode=count -coverprofile=build/coverage.out $(PACKAGES) | tee build/test.out
	go tool cover -html=build/coverage.out -o build/coverage.html
	go2xunit -input build/test.out -output build/test.xml
	! grep -e "--- FAIL" -e "^FAIL" build/test.out

bench:
	go test -bench .

vendor.update:
	dep ensure --update

vendor.ensure:
	dep ensure

### TOOLS

tools:
	go get -u github.com/golang/dep/cmd/dep
	go get -u github.com/tebeka/go2xunit
	go get -u golang.org/x/tools/cmd/cover
	go get -u github.com/golang/lint/golint

.PHONY: all build builddir vet lint clean fmt test testreport vendor.update vendor.ensure tools


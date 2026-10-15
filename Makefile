GO ?= go
export CGO_ENABLED := 0
.PHONY: build test vet lint integration
build:
	$(GO) build -trimpath -ldflags="-s -w" -o dist/pftw.exe ./cmd/pftw
	GOOS=windows $(GO) build -trimpath -ldflags="-s -w -H windowsgui" -o dist/pftww.exe ./cmd/pftww
test:
	$(GO) test ./...
vet:
	$(GO) vet ./...
lint:
	$(GO) run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./...
	$(GO) run github.com/kisielk/errcheck@v1.30.0 -ignoretests ./...
integration:
	PFTW_INTEGRATION=1 $(GO) test -tags integration ./...

.PHONY: resources
resources:
	$(GO) generate ./cmd/pftw

.PHONY: docs
docs: build
	./dist/pftw.exe docs > docs/cli.md

# Audit reachable vulnerabilities; no dependency files are changed.
.PHONY: security deps-update
security:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...

# Refresh dependencies, then verify the resulting module graph and application.
deps-update:
	$(GO) get -u ./...
	$(GO) mod tidy
	$(GO) run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
	$(GO) vet ./...
	$(GO) test ./...
	$(GO) run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./...
	$(GO) run github.com/kisielk/errcheck@v1.30.0 -ignoretests ./...
	$(GO) build -trimpath -ldflags="-s -w" -o dist/pftw.exe ./cmd/pftw
	GOOS=windows $(GO) build -trimpath -ldflags="-s -w -H windowsgui" -o dist/pftww.exe ./cmd/pftww

.PHONY: quality fuzz coverage
quality: security vet lint test build

fuzz:
	$(GO) test ./internal/pyver -run=FuzzParse -fuzz=FuzzParse -fuzztime=10s -parallel=2
	$(GO) test ./internal/archive -run=FuzzSafeName -fuzz=FuzzSafeName -fuzztime=10s -parallel=2

coverage: build
	$(GO) test -coverprofile=dist/coverage.out ./...
	$(GO) tool cover -func=dist/coverage.out

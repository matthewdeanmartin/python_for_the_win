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
integration:
	PFTW_INTEGRATION=1 $(GO) test -tags integration ./...

.PHONY: resources
resources:
	$(GO) generate ./cmd/pftw

.PHONY: docs
docs: build
	./dist/pftw.exe docs > docs/cli.md

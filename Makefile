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

# PyPI wheels carrying the Windows executables: make wheels VERSION=0.3.0 (a PEP 440 version).
# Publishing is a human step; see docs/pypi.md. wheels-check installs them with pip, wheels only.
WHEEL_ARCHES ?= amd64 arm64
PKG := github.com/matthewdeanmartin/python_for_the_win/internal/cli
.PHONY: wheels wheels-check
wheels:
	@test -n "$(VERSION)" || { echo "usage: make wheels VERSION=0.3.0" >&2; exit 2; }
	rm -rf dist/wheels dist/wheel-build
	set -e; for arch in $(WHEEL_ARCHES); do \
		GOOS=windows GOARCH=$$arch $(GO) build -trimpath -ldflags="-s -w -X $(PKG).Version=$(VERSION) -X $(PKG).Commit=$$(git rev-parse --short HEAD)" -o dist/wheel-build/$$arch/pftw.exe ./cmd/pftw; \
		GOOS=windows GOARCH=$$arch $(GO) build -trimpath -ldflags="-s -w -H windowsgui" -o dist/wheel-build/$$arch/pftww.exe ./cmd/pftww; \
		$(GO) run ./internal/wheel/cmd -version $(VERSION) -platform $$arch -pftw dist/wheel-build/$$arch/pftw.exe -pftww dist/wheel-build/$$arch/pftww.exe -out dist/wheels; \
	done
wheels-check: wheels
	set -e; tmp=$$(mktemp -d); trap 'rm -rf "$$tmp"' EXIT; \
	uv venv --quiet --seed "$$tmp/venv"; \
	"$$tmp/venv/Scripts/python.exe" -m pip install --quiet --only-binary :all: --no-index --find-links dist/wheels pftw==$(VERSION); \
	"$$tmp/venv/Scripts/pftw.exe" version | grep -F "pftw $(VERSION)"; \
	"$$tmp/venv/Scripts/python.exe" -m pftw version | grep -F "pftw $(VERSION)"; \
	"$$tmp/venv/Scripts/pftw.exe" self-update >/dev/null 2>&1 && code=0 || code=$$?; test "$$code" = 3; \
	"$$tmp/venv/Scripts/python.exe" -m pip uninstall --quiet --yes pftw; \
	test ! -e "$$tmp/venv/Scripts/pftw.exe"; \
	echo "wheels-check: pip install, pftw, python -m pftw, self-update refusal and uninstall passed"

# Changelog gate (keepachangelog-manager); settings in changelogmanager.toml.
KACLM ?= uvx --from keepachangelog-manager-fork==6.10.1 kaclm
.PHONY: changelog-check
changelog-check:
	$(KACLM) validate --strict

# Workflow gate: actionlint plus zizmor (online audits use the gh login when available).
# Upgrade and re-pin actions first with: GITHUB_TOKEN=$$(gh auth token) gha-update
ZIZMOR ?= uvx zizmor@1.31.0
.PHONY: workflows-check
workflows-check:
	actionlint .github/workflows/*.yml
	GH_TOKEN="$${GH_TOKEN:-$$(gh auth token 2>/dev/null)}" $(ZIZMOR) .github/workflows

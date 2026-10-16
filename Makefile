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
# Lint every CI OS: build-tagged files (e.g. *_other.go) are invisible to a Windows-only run.
LINT_OSES ?= windows linux darwin
# The linters are built for the host once; GOOS then selects which files they analyze.
TOOLS := $(CURDIR)/dist/tools
lint:
	GOBIN=$(TOOLS) $(GO) install honnef.co/go/tools/cmd/staticcheck@v0.8.2
	GOBIN=$(TOOLS) $(GO) install github.com/kisielk/errcheck@v1.30.0
	set -e; for os in $(LINT_OSES); do \
		echo "lint GOOS=$$os"; \
		GOOS=$$os $(TOOLS)/staticcheck ./...; \
		GOOS=$$os $(TOOLS)/errcheck -ignoretests ./...; \
	done
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
	$(MAKE) lint
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

# Reproducible workflow gate, including on untrusted PRs without credentials.
# Also run zizmor online during maintenance to check remote action metadata.
# Upgrade and re-pin actions first with: GITHUB_TOKEN=$$(gh auth token) gha-update
ZIZMOR ?= uvx zizmor@1.31.0
.PHONY: workflows-check
workflows-check:
	$(GO) run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 .github/workflows/*.yml
	$(ZIZMOR) --offline .github/workflows

# Release wheels wrap goreleaser's binaries (dist/release), so PyPI and the GitHub release ship
# identical executables: make release-wheels VERSION=0.1.0 (the tag; v and semver pre-releases are fine).
.PHONY: release-wheels
release-wheels:
	@test -n "$(VERSION)" || { echo "usage: make release-wheels VERSION=0.1.0" >&2; exit 2; }
	rm -rf dist/wheels
	set -e; for arch in amd64 arm64; do \
		manager=$$(ls -d dist/release/pftw_windows_$${arch}_*/ | head -n 1); \
		windowed=$$(ls -d dist/release/pftww_windows_$${arch}_*/ | head -n 1); \
		$(GO) run ./internal/wheel/cmd -version $(VERSION) -platform $$arch -pftw $${manager}pftw.exe -pftww $${windowed}pftww.exe -out dist/wheels; \
	done

# Documentation site (MkDocs, published on Read the Docs). Python 3.13 matches .readthedocs.yaml and
# has published wheels for every pinned dependency; nothing is built from source.
MKDOCS ?= uvx --no-build --python 3.13 --with-requirements docs/requirements.txt mkdocs
.PHONY: docs-check docs-serve
docs-check:
	$(MKDOCS) build --strict --site-dir dist/site
docs-serve:
	$(MKDOCS) serve

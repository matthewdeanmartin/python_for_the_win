#!/usr/bin/env bash
# Install pftw from PyPI in Git Bash on Windows, then hand over to pftw.
#
#   curl -LsSf https://raw.githubusercontent.com/matthewdeanmartin/python_for_the_win/main/install.sh | bash
#   curl -LsSf .../install.sh | bash -s -- install 3.15 --path none    # arguments go to pftw
#
# This script never edits PATH. It downloads the pftw wheel, checks its SHA-256 against the
# package index, extracts pftw.exe to a temporary folder and runs it. pftw installs Python and,
# if you agree, puts Python and itself on PATH. The temporary copy is deleted afterwards.
#
# Settings (environment variables):
#   PFTW_VERSION        pftw version to install, e.g. 0.3.0 (default: newest final release)
#   PFTW_INDEX_URL      PEP 503 simple index (default: https://pypi.org/simple); mirrors work
#   PFTW_KEEP           copy pftw.exe (and pftww.exe) into this folder instead of running it
#   PFTW_ALLOW_INSECURE set to 1 to allow a non-HTTPS index (testing only)

# Everything runs inside main, called on the last line, so a partial download runs nothing.
main() {
	set -euo pipefail

	local index="${PFTW_INDEX_URL:-https://pypi.org/simple}"
	index="${index%/}"
	local tab=$'\t'

	case "$(uname -s)" in
	MINGW* | MSYS* | CYGWIN*) ;;
	*) die "pftw installs Python on Windows. Run this from Git Bash on Windows, or use install.ps1." ;;
	esac
	case "$index" in
	https://*) ;;
	*) [ "${PFTW_ALLOW_INSECURE:-}" = 1 ] || die "PFTW_INDEX_URL must use https:// because the index supplies the checksum." ;;
	esac
	local arch="${PROCESSOR_ARCHITEW6432:-${PROCESSOR_ARCHITECTURE:-}}"
	local plat
	case "$arch" in
	AMD64 | amd64 | x86_64) plat=win_amd64 ;;
	ARM64 | arm64 | aarch64) plat=win_arm64 ;;
	*) die "Unsupported Windows architecture: ${arch:-unknown}" ;;
	esac
	need curl
	need sha256sum

	local page
	page="$(curl -fsSL "$index/pftw/")" || die "Cannot read $index/pftw/"
	# One "filename<TAB>href" line per wheel for this platform; yanked releases are skipped.
	local links
	links="$(printf '%s\n' "$page" | grep -o '<a [^>]*>[^<]*</a>' | grep -v 'data-yanked' |
		sed -n 's/.*href="\([^"]*\)".*>\(pftw-[^<]*-py3-none-'"$plat"'\.whl\)<\/a>/\2\t\1/p')" || true
	[ -n "$links" ] || die "No pftw wheel for $plat at $index"

	local version="${PFTW_VERSION:-}"
	version="${version#v}"
	if [ -z "$version" ]; then
		version="$(printf '%s\n' "$links" | sed -n 's/^pftw-\([^-]*\)-py3-.*/\1/p' |
			grep -E '^[0-9]+(\.[0-9]+)*$' | sort -t. -k1,1n -k2,2n -k3,3n -k4,4n | tail -n 1)" || true
		[ -n "$version" ] || die "No final pftw release for $plat at $index; set PFTW_VERSION to choose a pre-release."
	fi
	local line
	line="$(printf '%s\n' "$links" | grep -F "pftw-$version-py3-none-$plat.whl$tab" | head -n 1)" ||
		die "pftw $version has no wheel for $plat at $index"

	local href="${line#*"$tab"}"
	href="${href//&amp;/&}"
	local hash="${href##*#sha256=}"
	[ "$hash" != "$href" ] && [ "${#hash}" -eq 64 ] || die "The index gave no SHA-256 for pftw $version; refusing to install."
	local url="${href%%#*}"
	case "$url" in
	https://* | http://*) ;;
	/*) url="$(printf '%s' "$index" | sed -E 's#^(https?://[^/]+).*#\1#')$url" ;;
	*) url="$index/pftw/$url" ;;
	esac

	local tmp
	tmp="$(mktemp -d)"
	# shellcheck disable=SC2064 # Expand now: tmp is local to main.
	trap "rm -rf '$tmp'" EXIT
	curl -fsSL -o "$tmp/pftw.whl" "$url" || die "Download failed: $url"
	local actual
	actual="$(sha256sum "$tmp/pftw.whl" | cut -d' ' -f1)"
	[ "$actual" = "$(printf '%s' "$hash" | tr 'A-F' 'a-f')" ] || die "SHA-256 mismatch for $url; nothing was run."

	mkdir -p "$tmp/bin"
	if command -v unzip >/dev/null 2>&1; then
		unzip -q -j -o "$tmp/pftw.whl" '*.data/scripts/*' -d "$tmp/bin"
	else
		# Windows 10+ ships bsdtar, which reads ZIP files.
		mkdir -p "$tmp/x"
		"$(cygpath -u "$SYSTEMROOT")/System32/tar.exe" -xf "$(cygpath -w "$tmp/pftw.whl")" -C "$(cygpath -w "$tmp/x")"
		cp "$tmp"/x/*.data/scripts/*.exe "$tmp/bin/"
	fi
	[ -f "$tmp/bin/pftw.exe" ] || die "The wheel did not contain pftw.exe"
	echo "pftw $version ($plat) from $index, SHA-256 verified." >&2

	if [ -n "${PFTW_KEEP:-}" ]; then
		mkdir -p "$PFTW_KEEP"
		cp "$tmp"/bin/*.exe "$PFTW_KEEP/"
		echo "Saved pftw.exe to $PFTW_KEEP" >&2
		return 0
	fi

	# Interactive prompts must read the terminal, not this script arriving on stdin.
	local input=/dev/null
	if [ -t 2 ] && (exec </dev/tty) 2>/dev/null; then
		input=/dev/tty
	fi
	if [ $# -eq 0 ]; then
		if [ "$input" = /dev/tty ]; then
			set -- # No arguments: pftw opens its wizard.
		else
			set -- install
		fi
	fi
	local status=0
	"$tmp/bin/pftw.exe" "$@" <"$input" || status=$?
	if [ "$status" -eq 0 ] && [ -n "${LOCALAPPDATA:-}" ] && [ -f "$LOCALAPPDATA\\pftw\\shims\\pftw.exe" ]; then
		echo "pftw is now at $LOCALAPPDATA\\pftw\\shims\\pftw.exe. Open a new terminal to pick up PATH changes." >&2
	fi
	return "$status"
}

die() {
	echo "pftw installer: $*" >&2
	exit 1
}

need() {
	command -v "$1" >/dev/null 2>&1 || die "$1 is required"
}

main "$@"

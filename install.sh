#!/bin/sh
# Installs the latest Netlify Dynamic DNS (nddns) release from GitHub.
#
# POSIX sh on purpose: /bin/sh is dash on Debian and Raspberry Pi OS.
#   curl -fsSL https://raw.githubusercontent.com/oscartbeaumont/netlify-dynamic-dns/master/install.sh | sh
# Overrides: PREFIX (default /usr/local), DESTDIR, NDDNS_VERSION (release tag),
# NDDNS_OS, NDDNS_ARCH (amd64, 386, arm64, armv7, armv6), GITHUB_TOKEN.
set -u

REPO="oscartbeaumont/netlify-dynamic-dns"
PROGRAM="nddns"
INSTALL_URL="https://raw.githubusercontent.com/${REPO}/master/install.sh"

say() { printf '%s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

# Print every value of the "$1" field in the JSON on stdin. Splitting on commas
# puts one key/value pair per line, which is enough for the fields we read and
# avoids depending on jq.
json_strings() {
	tr ',' '\n' | sed -n 's/^[[:space:]]*"'"$1"'"[[:space:]]*:[[:space:]]*"\([^"]*\)"[[:space:]]*$/\1/p'
}

detect_os() {
	[ -n "${NDDNS_OS:-}" ] && { printf '%s\n' "$NDDNS_OS"; return 0; }
	kernel=$(uname -s) || die "could not run 'uname' to detect the operating system"
	case $kernel in
		Darwin) printf 'darwin\n' ;;
		Linux) printf 'linux\n' ;;
		*) die "operating system '$kernel' is not supported by this script.
Supported: Darwin, Linux. Set NDDNS_OS to override." ;;
	esac
}

SUPPORTED_ARCHS="amd64, 386, arm64, armv7, armv6"

detect_arch() {
	if [ -n "${NDDNS_ARCH:-}" ]; then
		case $NDDNS_ARCH in
			amd64 | 386 | arm64 | armv7 | armv6) printf '%s\n' "$NDDNS_ARCH"; return 0 ;;
			*) die "NDDNS_ARCH='${NDDNS_ARCH}' is not supported.
Supported architectures: ${SUPPORTED_ARCHS}." ;;
		esac
	fi

	machine=$(uname -m) || die "could not run 'uname' to detect the architecture"
	case $machine in
		x86_64 | amd64)          printf 'amd64\n' ;;
		i386 | i686 | i86pc)     printf '386\n' ;;
		aarch64 | arm64)         printf 'arm64\n' ;;
		armv7l | armv7 | armv8l) printf 'armv7\n' ;;
		armv6l)                  printf 'armv6\n' ;;
		*) die "architecture '$machine' is not supported by this script.
Supported architectures: ${SUPPORTED_ARCHS}. Set NDDNS_ARCH to override." ;;
	esac
}

tmp_dir=""
cleanup() {
	[ -z "$tmp_dir" ] || rm -rf "$tmp_dir"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

command -v curl >/dev/null 2>&1 || die "curl is required by this script but was not found.
Install it with your package manager (for example 'apt-get install curl') and try again."

os=$(detect_os) || exit 1
arch=$(detect_arch) || exit 1

if [ -n "${NDDNS_VERSION:-}" ]; then
	release_url="https://api.github.com/repos/${REPO}/releases/tags/${NDDNS_VERSION}"
else
	release_url="https://api.github.com/repos/${REPO}/releases/latest"
fi

fetch() {
	if [ -n "${GITHUB_TOKEN:-}" ]; then
		curl -fsSL -H "Authorization: Bearer ${GITHUB_TOKEN}" "$1"
	else
		curl -fsSL "$1"
	fi
}

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/nddns.XXXXXX") || die "could not create a temporary directory"
release_json="${tmp_dir}/release.json"
say "Looking up Netlify Dynamic DNS for ${os}/${arch}..."
fetch "$release_url" > "$release_json" || die "could not query the GitHub releases API (${release_url}).
If NDDNS_VERSION is set, check that the tag exists. Otherwise you may be rate
limited: set GITHUB_TOKEN to a personal access token and try again."

tag=$(json_strings tag_name < "$release_json" | head -n 1)
[ -n "$tag" ] || die "could not read the release tag from the GitHub API response"

# Match the exact asset basename, not a loose substring of the URL.
asset="${PROGRAM}_${tag}_${os}_${arch}"
if [ "$os" = "windows" ]; then
	asset="${asset}.exe"
fi
download_url=""
for url in $(json_strings browser_download_url < "$release_json"); do
	case $url in
		*/"$asset") download_url=$url; break ;;
	esac
done

if [ -z "$download_url" ]; then
	message="release ${tag} has no asset named '${asset}'.
Looked for: ${asset}
Assets in ${tag}:
$(json_strings browser_download_url < "$release_json" | sed -e 's|.*/|  |')"

	if [ -n "${NDDNS_OS:-}" ] || [ -n "${NDDNS_ARCH:-}" ]; then
		message="${message}
The NDDNS_OS/NDDNS_ARCH overrides are already set, so check that this release
publishes a binary for them."
	else
		message="${message}
Set NDDNS_OS or NDDNS_ARCH to override the detected platform (${os}/${arch}).
Supported architectures: ${SUPPORTED_ARCHS}."
	fi

	die "${message}
Releases before v3.0.3 use a different asset naming scheme and are not
supported by this script. Otherwise please open an issue at
https://github.com/${REPO}/issues"
fi

say "Downloading ${download_url}"
binary="${tmp_dir}/${PROGRAM}"
curl -fL --retry 3 --retry-delay 1 -o "$binary" "$download_url" || die "failed to download ${download_url}.
Try again, or download the binary manually from https://github.com/${REPO}/releases"

# A failed download can still leave an HTML error page or a truncated file
# behind, so check we are installing something binary-shaped.
size=$(wc -c < "$binary" | tr -d '[:space:]')
case $size in
	'' | *[!0-9]*) die "could not read the size of the downloaded file" ;;
esac
[ "$size" -ge 1024 ] || die "the download is empty or truncated (${size} bytes), refusing to install it"
[ "$(dd if="$binary" bs=1 count=1 2>/dev/null)" = "<" ] \
	&& die "the download looks like an HTML page, not a ${PROGRAM} binary (proxy or rate limit?)"
chmod +x "$binary" || die "could not make the downloaded binary executable"

prefix=${PREFIX:-/usr/local}
bindir="${DESTDIR:-}${prefix}/bin"
say "Installing to ${bindir}/${PROGRAM}"
if ! mkdir -p "$bindir" 2>/dev/null || ! cp "$binary" "${bindir}/${PROGRAM}" 2>/dev/null; then
	die "could not install to ${bindir}/${PROGRAM}.
Re-run this script with elevated privileges, for example:
  curl -fsSL ${INSTALL_URL} | sudo sh
Or install somewhere writable:
  curl -fsSL ${INSTALL_URL} | PREFIX=\"\$HOME/.local\" sh"
fi
chmod 755 "${bindir}/${PROGRAM}" 2>/dev/null || die "could not make ${bindir}/${PROGRAM} executable"

say "Netlify Dynamic DNS installed to: ${bindir}/${PROGRAM}"
case ":${PATH}:" in
	*":${bindir}:"*) ;;
	*) warn "${bindir} is not in your PATH; use the full path to run ${PROGRAM}" ;;
esac
say "Run '${PROGRAM} --version' to verify the installation."

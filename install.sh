#!/bin/sh
# Installs the latest release of Netlify Dynamic DNS from GitHub
# Repository: https://github.com/oscartbeaumont/netlify-dynamic-dns

function error() {
  echo $1 >&2
  exit 1
}

if test "$(uname)" = "Darwin"; then
    OS="darwin"
elif test "$(uname)" = "Linux"; then
    OS="linux"
else
    error "Operating system '$(uname)' is not supported by this script!"
fi

case $(uname -m) in
    i386 | i686)      architecture="386" ;;
    x86_64)           architecture="amd64" ;;
    arm64 | aarch64)  architecture="arm64" ;;
    armv7 | armv7l)   architecture="armv7" ;;
    *)                error "System architecture '$(uname -m)' is not supported by this script!" ;;
esac

download_url=$(curl -s https://api.github.com/repos/oscartbeaumont/netlify-dynamic-dns/releases/latest \
    | grep "browser_download_url.*nddns_.*_${OS}_${architecture}" \
    | cut -d ":" -f 2,3 \
    | tr -d \")

if [[ -z "${download_url// }" ]]; then
  error "Could not locate a binary for your operating system and architecture!"
else
  echo "Downloading binary from ${download_url// }"
fi;

temp_dir=$(mktemp -d -t nddns)
function cleanup {
  rm -rf $temp_dir
}
trap cleanup EXIT

curl -o "$temp_dir/nddns" -Ls $download_url || { error "Error getting latest binary from Github. Try again or open an issue on the repository!"; }
chmod +x "$temp_dir/nddns"
mv "$temp_dir/nddns" /usr/local/bin/ || { error "Error moving binary to /usr/local/bin. Maybe try running this script as root!"; }

echo "Netlify Dynamic DNS Installed to: $(which nddns)"

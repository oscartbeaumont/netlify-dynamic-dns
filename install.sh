#!/bin/sh
# Installs the latest release of Netlify Dynamic DNS from GitHub
# Repository: https://github.com/oscartbeaumont/netlify-dynamic-dns

if test "$(uname)" = "Darwin"; then
    OS="macOS"
elif test "$(uname)" = "Linux"; then
    OS="Linux"
else
    echo "Operating system '$(uname)' is not supported by this script!"
    exit 1;
fi

curl -s https://api.github.com/repos/oscartbeaumont/netlify-dynamic-dns/releases/latest \
    | grep "browser_download_url.*nddns_$OS\"" \
    | cut -d ":" -f 2,3 \
    | tr -d \" \
    | wget -O nddns -qi - || { rm -rf ./nddns; echo 'Error getting latest binary from Github. Try again or open an issue on the repository!'; exit 1; }

chmod +x nddns

mv nddns /usr/local/bin/ || { rm -rf ./nddns; echo 'Error moving binary to /usr/local/bin. Maybe try running this script as root!'; exit 1; }

echo "Netlify Dynamic DNS Installed to: $(which nddns)"

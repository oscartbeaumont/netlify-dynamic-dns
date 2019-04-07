#!/bin/bash

build_target=(
    darwin/386
    darwin/amd64
    linux/386
    linux/amd64
    linux/arm
    linux/arm64
    windows/386
    windows/amd64
)

mkdir dist/
for os_arch in "${build_target[@]}"
do
    goos=${os_arch%/*}
    goarch=${os_arch#*/}
    GOOS=${goos} GOARCH=${goarch} go build -o dist/netlify-ddns_${goos}_${goarch} ./cmd/
    echo "Built ${os_arch}"
done
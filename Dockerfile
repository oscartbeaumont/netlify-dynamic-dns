# syntax=docker/dockerfile:1
#
# Multi-arch image for Netlify Dynamic DNS.
#
# The builder stage always runs on the build platform ($BUILDPLATFORM) and
# cross-compiles to the requested target, so building linux/amd64, linux/arm64
# and linux/arm/v7 only needs QEMU for the final image assembly, not for Go.

FROM --platform=$BUILDPLATFORM golang:alpine AS builder

ARG VERSION
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

RUN adduser -D -g '' nddns_usr

RUN apk update && \
    apk add --no-cache git ca-certificates && \
    update-ca-certificates

WORKDIR /src
COPY . .

# GOARM is only read by the Go toolchain when GOARCH=arm, so passing it for
# amd64/arm64 targets is harmless. TARGETVARIANT is "v7" for linux/arm/v7 and
# empty elsewhere, hence the #v stripping to get GOARM=7.
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} \
    go build -ldflags "-X main.Version=${VERSION}" -o ./nddns ./cmd

FROM scratch
LABEL org.opencontainers.image.title="Netlify Dynamic DNS"
LABEL org.opencontainers.image.source="https://github.com/oscartbeaumont/netlify-dynamic-dns"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
USER nddns_usr
COPY --from=builder /src/nddns /
ENTRYPOINT ["/nddns"]

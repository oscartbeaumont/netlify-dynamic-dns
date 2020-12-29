FROM golang:alpine AS builder

ARG VERSION

RUN adduser -D -g '' nddns_usr

RUN apk update && \
    apk add --no-cache git ca-certificates && \
    update-ca-certificates

WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-X main.Version=${VERSION}" -o ./nddns ./cmd

FROM scratch
LABEL org.opencontainers.image.title Netlify Dynamic DNS
LABEL org.opencontainers.image.source https://github.com/oscartbeaumont/netlify-dynamic-dns
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
USER nddns_usr
COPY --from=builder /src/nddns /
ENTRYPOINT ["/nddns"]
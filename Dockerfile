FROM golang:alpine AS builder

RUN adduser -D -g '' app

RUN apk update && \
    apk add --no-cache git ca-certificates && \
    update-ca-certificates

RUN wget https://raw.githubusercontent.com/golang/dep/master/install.sh && \
    sh install.sh

WORKDIR $GOPATH/src/github.com/oscartbeaumont/netlify-dynamic-dns
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure --vendor-only
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o ./netlify-ddns ./cmd

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
USER app
COPY --from=builder /go/src/github.com/oscartbeaumont/netlify-dynamic-dns/netlify-ddns ./netlify-ddns
ENTRYPOINT ["./netlify-ddns"]
FROM golang:alpine as builder
RUN apk add git
RUN go get -u github.com/pkg/errors
RUN go get github.com/oscartbeaumont/netlify-dynamic-dns
WORKDIR /go/src/github.com/github.com/oscartbeaumont/netlify-dynamic-dns
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o netlify-dynamic-dns ./cmd

FROM oscartbeaumont/scratch
COPY --from=builder /go/src/github.com/github.com/oscartbeaumont/netlify-dynamic-dns/netlify-dynamic-dns /
CMD ["/netlify-dynamic-dns"]

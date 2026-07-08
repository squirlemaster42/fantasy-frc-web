# Multi-stage build: compile goose from source to avoid registry rate limits.
FROM golang:1.26 AS goose-builder
RUN CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.27.1

FROM alpine:3.19
RUN apk add --no-cache ca-certificates

COPY --from=goose-builder /go/bin/goose /usr/local/bin/goose
COPY migrations /migrations

ENTRYPOINT ["goose", "-dir", "/migrations"]

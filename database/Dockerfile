# Multi-stage build: pull official goose binary, create minimal image
FROM ghcr.io/pressly/goose:v3.27.1 AS goose

FROM alpine:3.19
RUN apk add --no-cache ca-certificates

COPY --from=goose /usr/local/bin/goose /usr/local/bin/goose
COPY migrations /migrations

ENTRYPOINT ["goose", "-dir", "/migrations"]

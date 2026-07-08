# Build stage
FROM golang:1.26 AS builder

WORKDIR /app

# Download Tailwind CSS CLI
RUN curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 && \
    chmod +x tailwindcss-linux-x64 && \
    mv tailwindcss-linux-x64 tailwindcss

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Generate templ files and build CSS using the templ version from go.mod
RUN go tool templ generate && \
    ./tailwindcss -i ./assets/input.css -o ./assets/css/styles.css --minify

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# Runtime stage
FROM alpine:3.22

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/assets ./assets

# Optional: document the port
EXPOSE 8080

CMD ["./server"]

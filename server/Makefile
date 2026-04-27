# CSS build commands
TAILWIND_BIN := ./tailwindcss
CSS_INPUT := ./assets/input.css
CSS_OUTPUT := ./assets/css/styles.css

# Download Tailwind CLI if not present
$(TAILWIND_BIN):
	@echo "Downloading Tailwind CSS CLI..."
	@curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64
	@chmod +x tailwindcss-linux-x64
	@mv tailwindcss-linux-x64 tailwindcss
	@echo "Tailwind CSS CLI downloaded successfully!"

# Build CSS for production (minified)
build-css: $(TAILWIND_BIN)
	$(TAILWIND_BIN) -i $(CSS_INPUT) -o $(CSS_OUTPUT) --minify
	@echo "CSS built successfully!"

# Watch CSS for development
watch-css: $(TAILWIND_BIN)
	$(TAILWIND_BIN) -i $(CSS_INPUT) -o $(CSS_OUTPUT) --watch

# Run development server with hot reload
run-verbose: $(TAILWIND_BIN)
	@echo "Starting development server with verbose logging..."
	@make watch-css &
	@go tool templ generate --watch --proxy="http://localhost:3000" --cmd="go run . -v -skipScoring=true -log-format=text"

# Generate templ files
generate:
	go tool templ generate

# Production build (static binary with embedded assets)
build: build-css generate
	CGO_ENABLED=0 go build -ldflags="-s -w" -o server .
	@echo "Server binary built successfully!"

# Build for deployment (cross-compile for Linux amd64)
build-linux: build-css generate
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o server .
	@echo "Linux binary built successfully!"

# Run database migrations
migrate:
	@echo "Running migrations..."
	@./../deploy/scripts/migrate.sh

# Install production binary to deploy location
install: build
	@cp server ../deploy/ansible/files/ 2>/dev/null || mkdir -p ../deploy/ansible/files && cp server ../deploy/ansible/files/
	@echo "Binary copied to deploy/ansible/files/"

# Full production build
prod: build
	@echo "Production build complete. Binary: server"

.PHONY: build-css watch-css run build build-linux migrate install prod generate

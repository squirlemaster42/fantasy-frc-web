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
run: $(TAILWIND_BIN)
	@echo "Starting development server..."
	@make watch-css &
	@go tool templ generate --watch --proxy="http://localhost:3000" --cmd="go run . -skipScoring=true -populateTeams=false"

# Production build
build: build-css
	go build -o server .

.PHONY: build-css watch-css run build

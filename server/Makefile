run:
	@go tool templ generate --watch --proxy="http://localhost:3000" --cmd="go run . -skipScoring=true -populateTeams=false"
	@go run .

test-csrf:
	@echo "Running CSRF protection tests..."
	@go test -v ./... -run TestCSRF

test-csrf-manual:
	@echo "Running manual CSRF tests..."
	@../scripts/test_csrf.sh

test:
	@go test -v ./...

.PHONY: run test-csrf test-csrf-manual test

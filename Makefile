.PHONY: build install test test-unit test-integration test-e2e test-race test-coverage clean run fmt lint record-gifs record-gif

fmt:
	gofmt -w .

lint: fmt
	go vet ./...

build: lint
	go build -o bin/grund .

install:
	go install .

test: test-unit test-integration test-e2e test-race test-coverage

test-unit:
	go test ./internal/domain/... -v

test-integration:
	go test ./internal/application/... -v

test-e2e:
	go test ./test/integration/... -v

test-race:
	go test -race ./...

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

run:
	go run .

# GIF Recording (requires VHS: brew install charmbracelet/tap/vhs)
# The script handles all prerequisites automatically:
#   - Checks Docker is running
#   - Verifies grund is installed
#   - Starts/stops demo services as needed

record-gifs:
	@./scripts/record-gifs.sh

record-gif:
ifndef TAPE
	$(error TAPE is required. Usage: make record-gif TAPE=grund-up)
endif
	@./scripts/record-gifs.sh $(TAPE)

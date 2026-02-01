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
record-gifs:
	@echo "Recording all GIFs..."
	@mkdir -p docs/assets
	@for tape in scripts/recordings/*.tape; do \
		name=$$(basename "$$tape" .tape); \
		echo "Recording $$name..."; \
		vhs "$$tape" -o "docs/assets/$$name.gif"; \
	done
	@echo "All GIFs recorded to docs/assets/"

record-gif:
ifndef TAPE
	$(error TAPE is required. Usage: make record-gif TAPE=grund-up)
endif
	@mkdir -p docs/assets
	vhs scripts/recordings/$(TAPE).tape -o docs/assets/$(TAPE).gif

.PHONY: build test install uninstall start stop restart status logs logs-err \
       build-docker start-docker stop-docker restart-docker logs-docker \
       docker-push lint strict-lint swagger build-all build-release

APP_NAME := mmoney
PORT ?= 9220
BASE_PATH ?= /apps/ext/mmoney/
PLIST_NAME := com.localitas.app.mmoney
PLIST_FILE := $(HOME)/Library/LaunchAgents/$(PLIST_NAME).plist
LOG_DIR := $(HOME)/.localitas/logs/mmoney
BIN_PATH := $(shell pwd)/bin/mmoney-server
WORK_DIR := $(shell pwd)
VERSION ?= 1.0.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)"

# ── Build & Test ──────────────────────────────────────────────

build: lint

build-linux: lint
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath \
		-o mmoney-server-linux-amd64 ./cmd/mmoney-server
	@mkdir -p bin
	go build -o bin/mmoney-server ./cmd/mmoney-server

test: lint
	CGO_ENABLED=1 go test -tags fts5 -v ./...

lint:
	@echo "Running gofmt..."
	@gofmt -w .
	@echo "Running go vet..."
	@go vet ./...

strict-lint: lint
	@echo "Running staticcheck..."
	@if ! command -v staticcheck > /dev/null 2>&1; then \
		echo "Installing staticcheck..."; \
		go install honnef.co/go/tools/cmd/staticcheck@latest; \
	fi
	@staticcheck ./...
	@echo "staticcheck passed"

swagger:
	@curl -s http://localhost:$(PORT)/swagger.json | python3 -m json.tool

# ── Cross-compile ────────────────────────────────────────────

build-all:
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -trimpath -o bin/mmoney-server-linux-amd64 ./cmd/mmoney-server
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -trimpath -o bin/mmoney-server-linux-arm64 ./cmd/mmoney-server
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -trimpath -o bin/mmoney-server-darwin-arm64 ./cmd/mmoney-server
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -trimpath -o bin/mmoney-server-darwin-amd64 ./cmd/mmoney-server

build-release:
	@echo "Building obfuscated release binaries..."
	@if ! command -v garble > /dev/null 2>&1; then \
		echo "Installing garble..."; \
		go install mvdan.cc/garble@latest; \
	fi
	@mkdir -p bin
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 GOGARBLE=github.com/localitas garble -literals -tiny build $(LDFLAGS) -trimpath -o bin/mmoney-server-darwin-arm64 ./cmd/mmoney-server
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 GOGARBLE=github.com/localitas garble -literals -tiny build $(LDFLAGS) -trimpath -o bin/mmoney-server-darwin-amd64 ./cmd/mmoney-server
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOGARBLE=github.com/localitas garble -literals -tiny build $(LDFLAGS) -trimpath -o bin/mmoney-server-linux-amd64 ./cmd/mmoney-server
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 GOGARBLE=github.com/localitas garble -literals -tiny build $(LDFLAGS) -trimpath -o bin/mmoney-server-linux-arm64 ./cmd/mmoney-server
	@echo "Release binaries built:"
	@ls -lh bin/mmoney-server-*

# ── Native (launchd) ─────────────────────────────────────────

install: build
	@mkdir -p $(LOG_DIR)
	@sed 's|$${BIN_PATH}|$(BIN_PATH)|g; s|$${PORT}|$(PORT)|g; s|$${BASE_PATH}|$(BASE_PATH)|g; s|$${LOG_DIR}|$(LOG_DIR)|g; s|$${WORK_DIR}|$(WORK_DIR)|g' \
		plist.template > $(PLIST_FILE)
	@echo "Installed launchd service: $(PLIST_NAME)"

uninstall: stop
	@rm -f $(PLIST_FILE)
	@echo "Uninstalled launchd service: $(PLIST_NAME)"

start: install
	@launchctl load $(PLIST_FILE) 2>/dev/null || true
	@echo "Started $(PLIST_NAME) on port $(PORT)"

stop:
	@launchctl unload $(PLIST_FILE) 2>/dev/null || true
	@echo "Stopped $(PLIST_NAME)"

restart: stop start

status:
	@launchctl list | grep $(PLIST_NAME) || echo "$(PLIST_NAME) is not running"

logs:
	@tail -f $(LOG_DIR)/stdout.log

logs-err:
	@tail -f $(LOG_DIR)/stderr.log

# ── Docker ────────────────────────────────────────────────────

build-docker: build-linux
	docker build -t mmoney:latest .

start-docker: build-docker stop-docker
	@docker run -d -p $(PORT):8000 --name mmoney \
		--log-opt max-size=10m --log-opt max-file=7 \
		mmoney:latest
	@echo "Waiting for mmoney to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		curl -sf http://localhost:$(PORT)/health.json > /dev/null 2>&1 && break; \
		sleep 1; \
	done
	@echo "mmoney running in Docker on port $(PORT)"

stop-docker:
	@docker rm -f mmoney 2>/dev/null || true

restart-docker: stop-docker start-docker

logs-docker:
	@docker logs -f mmoney

# ── Docker Registry (ghcr.io) ────────────────────────────────

GHCR_IMAGE := ghcr.io/localitas/localitas-app-$(APP_NAME)

docker-push: test build-docker
	docker tag $(APP_NAME):latest $(GHCR_IMAGE):latest
	docker tag $(APP_NAME):latest $(GHCR_IMAGE):$(VERSION)
	docker push $(GHCR_IMAGE):latest
	docker push $(GHCR_IMAGE):$(VERSION)
	@echo "✅ Pushed $(GHCR_IMAGE):latest and $(GHCR_IMAGE):$(VERSION)"

# ── Release ───────────────────────────────────────────────────

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

build-release: lint
	@mkdir -p dist
	@echo "Building $(APP_NAME) $(VERSION) ($(COMMIT))..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)" -trimpath \
		-o dist/mmoney-server-darwin-arm64 ./cmd/mmoney-server
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)" -trimpath \
		-o dist/mmoney-server-darwin-amd64 ./cmd/mmoney-server
	@echo "Built: dist/mmoney-server-darwin-arm64, dist/mmoney-server-darwin-amd64"

release: build-release
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "dev" ]; then echo "Set VERSION=vX.Y.Z"; exit 1; fi
	@echo "Creating release $(VERSION) on GitHub..."
	gh release create $(VERSION) \
		dist/mmoney-server-darwin-arm64 \
		dist/mmoney-server-darwin-amd64 \
		--title "$(VERSION)" --generate-notes
	@echo "✅ Released $(VERSION)"

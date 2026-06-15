BINARY     := gwsm
CMD        := ./cmd/gwsm
INSTALL_BIN := $(HOME)/.local/bin
SERVICE_DIR := $(HOME)/.config/systemd/user
VERSION    ?= 1.0.0

.PHONY: all build test lint install install-service enable-service deb tar dist clean

all: build

## build: compile the gwsm binary
build:
	go build -o $(BINARY) $(CMD)

## test: run all unit tests with coverage report
test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

## lint: run go vet (add golangci-lint if available)
lint:
	go vet ./...
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed, skipping"

## install: build and install binary to ~/.local/bin
install: build
	mkdir -p $(INSTALL_BIN)
	cp $(BINARY) $(INSTALL_BIN)/$(BINARY)
	@echo "Installed to $(INSTALL_BIN)/$(BINARY)"

## install-service: install the systemd user service unit
install-service:
	mkdir -p $(SERVICE_DIR)
	cp gwsm.service $(SERVICE_DIR)/gwsm.service
	systemctl --user daemon-reload
	@echo "Installed $(SERVICE_DIR)/gwsm.service"

## enable-service: install and start the service on login
enable-service: install install-service
	systemctl --user enable --now gwsm.service
	@echo "gwsm daemon enabled and started"

## deb: build a .deb package via Docker (output: dist/gwsm_VERSION_ARCH.deb)
deb:
	mkdir -p dist
	docker build \
		--target packager \
		--build-arg VERSION=$(VERSION) \
		-f Dockerfile \
		-t gwsm-deb:$(VERSION) .
	docker run --rm \
		-v $(CURDIR)/dist:/output \
		gwsm-deb:$(VERSION)
	@echo ""
	@echo "Package ready:"
	@ls dist/gwsm_*.deb

## tar: build a .tar.gz archive via Docker (output: dist/gwsm_VERSION_linux_ARCH.tar.gz)
tar:
	mkdir -p dist
	docker build \
		--target archiver \
		--build-arg VERSION=$(VERSION) \
		-f Dockerfile \
		-t gwsm-tar:$(VERSION) .
	docker run --rm \
		-v $(CURDIR)/dist:/output \
		gwsm-tar:$(VERSION)
	@echo ""
	@echo "Archive ready:"
	@ls dist/gwsm_*.tar.gz

## dist: build both .deb and .tar.gz
dist: deb tar

## clean: remove build artifacts
clean:
	rm -f $(BINARY) coverage.out
	rm -rf dist/

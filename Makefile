BINARY_NAME := yasp
BUILD_DIR   := build
BIN         := $(BUILD_DIR)/$(BINARY_NAME)

PREFIX      ?= /usr
DESTDIR     ?=
BINDIR      := $(DESTDIR)$(PREFIX)/bin

GO          ?= go
GO_SOURCES  := $(shell find cmd internal -name '*.go') go.mod go.sum

.PHONY: all build windows vet install uninstall clean help

all: build

## build: Build YASP into ./build/
build: $(BIN)

$(BIN): $(GO_SOURCES)
	$(GO) build -v -o $(BIN) ./cmd/yasp/

## windows: Cross-compile a Windows binary into ./build/
windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GO) build -v -o $(BUILD_DIR)/$(BINARY_NAME).exe ./cmd/yasp/

## vet: Run go vet
vet:
	$(GO) vet ./...

## install: Install the binary (run `make` first, then `sudo make install`)
install: $(BIN)
	install -Dm755 $(BIN) $(BINDIR)/$(BINARY_NAME)

## uninstall: Remove the installed binary
uninstall:
	rm -f $(BINDIR)/$(BINARY_NAME)

## clean: Remove build output
clean:
	rm -rf $(BUILD_DIR)

## help: List targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'

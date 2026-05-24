# Blsqui CLI - High Grade Multi-Platform Build Pipeline
VERSION=v1.0.0
CFLAGS=-std=c11 -D_GNU_SOURCE -D_POSIX_C_SOURCE=199309L -Wno-maybe-uninitialized -Wno-unused-variable

# Default target when you just run 'make'
.PHONY: all
all: clean build-mac build-linux build-windows

.PHONY: build-mac
build-mac:
	@echo "🍏 Building Mac Air Binary (Native ARM64)..."
	@mkdir -p dist
	CGO_ENABLED=1 CGO_CFLAGS="$(CFLAGS)" GOOS=darwin GOARCH=arm64 go build -o dist/blsqui-cli-darwin-arm64 main.go

.PHONY: build-linux
build-linux:
	@echo "🐧 Building Linux Server Binary..."
	@mkdir -p dist
	# Leverages a standard Go container to guarantee the correct Linux C-compiler environment
	docker run --rm -v $(CURDIR):/app -w /app golang:1.22-bookworm sh -c \
		"apt-get update && apt-get install -y gcc && CGO_ENABLED=1 CGO_CFLAGS='$(CFLAGS)' GOOS=linux GOARCH=amd64 go build -o dist/blsqui-cli-linux-amd64 main.go"

.PHONY: build-windows
build-windows:
	@echo "🪟 Building Windows Enterprise Binary..."
	@mkdir -p dist
	# Leverages mingw-w64 inside a container to cross-compile Windows CGO without leaving your Mac
	docker run --rm -v $(CURDIR):/app -w /app golang:1.22-bookworm sh -c \
		"apt-get update && apt-get install -y gcc-mingw-w64 && CGO_ENABLED=1 CGO_CFLAGS='$(CFLAGS)' GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -o dist/blsqui-cli-windows-amd64.exe main.go"

.PHONY: clean
clean:
	@echo "🧹 Cleaning up old builds..."
	rm -rf dist/*
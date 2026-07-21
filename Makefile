BINARY := windotp
VERSION ?= dev
PREFIX ?= /usr/local

.PHONY: build test install clean

build:
	mkdir -p bin
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/$(BINARY) ./cmd/windotp

test:
	go test ./...

install: build
	install -d "$(DESTDIR)$(PREFIX)/bin"
	install -m 0755 bin/$(BINARY) "$(DESTDIR)$(PREFIX)/bin/$(BINARY)"

clean:
	rm -rf bin dist

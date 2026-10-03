BIN := bin/hex-tui
PREFIX := $(HOME)/.local/bin

.PHONY: all build test run fmt vet install clean

all: build

build:
	go build -o $(BIN) .

test:
	go test ./...

run: build
	./$(BIN)

fmt:
	gofmt -w .

vet:
	go vet ./...

install: build
	install -Dm755 $(BIN) $(PREFIX)/hex-tui
	@echo "installé dans $(PREFIX)/hex-tui"

clean:
	rm -rf bin
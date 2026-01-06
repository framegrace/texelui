.PHONY: all build demos test tidy clean

GO ?= go
BINDIR ?= bin

TEXELUI_BIN := $(BINDIR)/texelui
DEMO_BIN := $(BINDIR)/texelui-demo

all: build demos test

build:
	$(GO) build ./...

demos: $(TEXELUI_BIN) $(DEMO_BIN)

$(BINDIR):
	mkdir -p $(BINDIR)

$(TEXELUI_BIN): $(BINDIR)
	$(GO) build -o $@ ./cmd/texelui

$(DEMO_BIN): $(BINDIR)
	$(GO) build -o $@ ./cmd/texelui-demo

test:
	$(GO) test ./...

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BINDIR)

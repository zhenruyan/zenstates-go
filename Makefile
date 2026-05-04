# Makefile for amdcpu-overclocking (Go version)

BINDIR := bin

.PHONY: all build clean install

all: build

build:
	go build -o $(BINDIR)/zenstates ./cmd/zenstates
	go build -o $(BINDIR)/togglecode ./cmd/togglecode

install:
	install -m 755 $(BINDIR)/zenstates /usr/local/bin/zenstates
	install -m 755 $(BINDIR)/togglecode /usr/local/bin/togglecode

clean:
	rm -rf $(BINDIR)

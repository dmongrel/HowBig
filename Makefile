# Makefile for HowBig
SHELL := /bin/bash
# Prerequisites
export CGO_ENABLED=1

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=HowBig.exe

# Define variables for easy maintenance
ISCC="/c/Program Files/Inno Setup 7/ISCC.exe"
SCRIPT="./innosetup.iss"

all: build

build:
	echo "Building project..."
	$(GOBUILD) -o $(BINARY_NAME) -v

run:
	$(GORUN) .

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

release: build
	echo "Packaging installer..."
	$(ISCC) $(SCRIPT)

.PHONY: all build run clean release

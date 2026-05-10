APP_NAME := tidyfs
MAIN := ./src/main.go

BUILD_DIR := bin
APP_BIN := $(BUILD_DIR)/$(APP_NAME)

VENV := .venv
VENV_PYTHON := $(VENV)/bin/python
VENV_PIP := $(VENV)/bin/pip

REQ_FILE := requirements.txt

GO := go
PYTHON_SYSTEM := python3

.PHONY: all deps go-deps py-deps build run dev clean py-clean clean-all rebuild help

all: deps build

deps: go-deps py-deps

go-deps:
	$(GO) mod tidy
	$(GO) mod download

$(VENV_PYTHON):
	$(PYTHON_SYSTEM) -m venv $(VENV)

py-venv: $(VENV_PYTHON)

py-deps: $(VENV_PYTHON)
	$(VENV_PIP) install --upgrade pip
	$(VENV_PIP) install -r $(REQ_FILE)

build:
	mkdir -p $(BUILD_DIR)
	$(GO) build -o $(APP_BIN) $(MAIN)

run: build
	./$(APP_BIN)

dev: deps run

rebuild: clean all

clean:
	rm -rf $(BUILD_DIR)

py-clean:
	rm -rf $(VENV)

clean-all: clean py-clean
	rm -rf classifier/__pycache__
	rm -rf extractor/__pycache__
	rm -rf files/*.json

help:
	@echo "TidyFS Makefile"
	@echo ""
	@echo "Commands:"
	@echo "  make              install deps and build"
	@echo "  make deps         install Go and Python deps"
	@echo "  make go-deps      run go mod tidy/download"
	@echo "  make py-venv      create Python virtualenv"
	@echo "  make py-deps      install Python deps"
	@echo "  make build        build Go binary into bin/tidyfs"
	@echo "  make run          build and run app"
	@echo "  make dev          install deps, build and run"
	@echo "  make clean        remove bin/"
	@echo "  make py-clean     remove .venv/"
	@echo "  make clean-all    remove bin, venv, pycache and generated json"
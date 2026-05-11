#!/usr/bin/env bash

set -euo pipefail

APP_NAME="tidyfs"
INSTALL_DIR="/usr/local/bin"

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_PATH="${PROJECT_DIR}/bin/${APP_NAME}"
INSTALL_PATH="${INSTALL_DIR}/${APP_NAME}"

echo "Installing ${APP_NAME}..."

if ! command -v make >/dev/null 2>&1; then
  echo "Error: make is not installed or not in PATH"
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Error: Go is not installed or not in PATH"
  exit 1
fi

cd "${PROJECT_DIR}"

echo "Building with Makefile..."
make build

if [ ! -f "${BINARY_PATH}" ]; then
  echo "Error: binary not found: ${BINARY_PATH}"
  exit 1
fi

chmod +x "${BINARY_PATH}"

echo "Installing to ${INSTALL_PATH}..."

if [ -w "${INSTALL_DIR}" ]; then
  cp "${BINARY_PATH}" "${INSTALL_PATH}"
else
  sudo cp "${BINARY_PATH}" "${INSTALL_PATH}"
fi

echo "Done."
echo "Now you can run:"
echo "  ${APP_NAME}"
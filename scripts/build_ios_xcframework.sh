#!/usr/bin/env bash
set -euo pipefail

TARGET="${1:-ios}"
OUT_FILE="${2:-app/ios/Frameworks/Nodemobile.xcframework}"
GOMOBILE_VERSION="${GOMOBILE_VERSION:-v0.0.0-20260217195705-b56b3793a9c4}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODULE_DIR="${REPO_ROOT}/nodemobile"
OUT_PATH="${REPO_ROOT}/${OUT_FILE}"

echo "Build ClipboardNode iOS XCFramework via gomobile"
echo "  RepoRoot : ${REPO_ROOT}"
echo "  ModuleDir: ${MODULE_DIR}"
echo "  Target   : ${TARGET}"
echo "  OutFile  : ${OUT_PATH}"
echo "  Gomobile : golang.org/x/mobile/cmd/gomobile@${GOMOBILE_VERSION}"

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "iOS gomobile binding requires macOS and Xcode." >&2
  exit 1
fi

if [[ ! -d "${MODULE_DIR}" ]]; then
  echo "nodemobile module not found: ${MODULE_DIR}" >&2
  exit 1
fi

mkdir -p "$(dirname "${OUT_PATH}")"

echo "Installing pinned gomobile..."
go install "golang.org/x/mobile/cmd/gomobile@${GOMOBILE_VERSION}"

export GOWORK=off

pushd "${MODULE_DIR}" >/dev/null
gomobile init
gomobile bind -target "${TARGET}" -prefix MFH -o "${OUT_PATH}" .
popd >/dev/null

if [[ ! -d "${OUT_PATH}" ]]; then
  echo "XCFramework not generated: ${OUT_PATH}" >&2
  exit 1
fi

echo "OK: ${OUT_PATH}"

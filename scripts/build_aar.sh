#!/usr/bin/env bash
set -euo pipefail

TARGET="${1:-android/arm64,android/arm,android/amd64,android/386}"
JAVA_PKG="${2:-com.myflowhub.gomobile}"
OUT_FILE="${3:-app/android/app/libs/myflowhub.aar}"
ANDROID_API="${ANDROID_API:-26}"
GOMOBILE_VERSION="${GOMOBILE_VERSION:-v0.0.0-20260217195705-b56b3793a9c4}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODULE_DIR="${REPO_ROOT}/nodemobile"
OUT_PATH="${REPO_ROOT}/${OUT_FILE}"

echo "Build ClipboardNode AAR via gomobile"
echo "  RepoRoot  : ${REPO_ROOT}"
echo "  ModuleDir : ${MODULE_DIR}"
echo "  Target    : ${TARGET}"
echo "  AndroidApi: ${ANDROID_API}"
echo "  JavaPkg   : ${JAVA_PKG}"
echo "  OutFile   : ${OUT_PATH}"
echo "  Gomobile  : golang.org/x/mobile@${GOMOBILE_VERSION}"

mkdir -p "$(dirname "${OUT_PATH}")"

go_bin="$(go env GOPATH)/bin"
case ":${PATH}:" in
  *":${go_bin}:"*) ;;
  *) export PATH="${go_bin}:${PATH}" ;;
esac

echo "Installing pinned gomobile..."
go install "golang.org/x/mobile/cmd/gomobile@${GOMOBILE_VERSION}"
go install "golang.org/x/mobile/cmd/gobind@${GOMOBILE_VERSION}"

export GOWORK=off

pushd "${MODULE_DIR}" >/dev/null
gomobile init
gomobile bind -target "${TARGET}" -androidapi "${ANDROID_API}" -javapkg "${JAVA_PKG}" -o "${OUT_PATH}" .
popd >/dev/null

echo "OK: ${OUT_PATH}"

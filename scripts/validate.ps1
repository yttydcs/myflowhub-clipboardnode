param(
  [switch]$SkipFlutter,
  [string]$FlutterRoot = $env:FLUTTER_ROOT,
  [string]$FlutterBin
)

$ErrorActionPreference = "Stop"

$env:GOWORK = "off"
New-Item -ItemType Directory -Force -Path "build" | Out-Null
go test ./... -count=1
go build -o "build/clipboardnode.exe" ./cmd/clipboardnode
go build -o "build/clipboardnode-bridge.exe" ./cmd/clipboardnode-bridge

if (-not $SkipFlutter) {
  if ($FlutterBin) {
    if (-not (Test-Path $FlutterBin)) {
      throw "FlutterBin does not exist: $FlutterBin"
    }
    $flutterCommand = $FlutterBin
  } elseif ($FlutterRoot) {
    $candidate = Join-Path $FlutterRoot "bin/flutter.bat"
    if (-not (Test-Path $candidate)) {
      throw "FlutterRoot does not contain bin/flutter.bat: $FlutterRoot"
    }
    $flutterCommand = $candidate
  } else {
    $flutter = Get-Command flutter -ErrorAction SilentlyContinue
    if ($flutter) {
      $flutterCommand = $flutter.Source
    }
  }

  if (-not $flutterCommand) {
    throw "flutter was not found. Add Flutter to PATH, set FLUTTER_ROOT, pass -FlutterRoot/-FlutterBin, or pass -SkipFlutter for Go-only validation."
  }

  if ($FlutterRoot -and -not $env:PUB_CACHE) {
    $env:PUB_CACHE = Join-Path $FlutterRoot ".pub-cache"
  }

  & $flutterCommand --version

  Push-Location app
  try {
    & $flutterCommand analyze
    & $flutterCommand test
  } finally {
    Pop-Location
  }
}
git diff --check

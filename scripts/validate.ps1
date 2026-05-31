$ErrorActionPreference = "Stop"

$env:GOWORK = "off"
New-Item -ItemType Directory -Force -Path "build" | Out-Null
go test ./... -count=1
go build -o "build/clipboardnode.exe" ./cmd/clipboardnode
git diff --check

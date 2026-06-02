param(
  [string]$Target = 'android/arm64,android/arm,android/amd64,android/386',
  [string]$JavaPkg = 'com.myflowhub.gomobile',
  [string]$OutFile = 'app/android/app/libs/myflowhub.aar',
  [int]$AndroidApi = 26,
  [string]$GomobileVersion = 'v0.0.0-20260217195705-b56b3793a9c4'
)

$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path | Split-Path -Parent
$moduleDir = Join-Path $repoRoot 'nodemobile'
$outPath = Join-Path $repoRoot $OutFile

Write-Host "Build ClipboardNode AAR via gomobile" -ForegroundColor Cyan
Write-Host "  RepoRoot  : $repoRoot"
Write-Host "  ModuleDir : $moduleDir"
Write-Host "  Target    : $Target"
Write-Host "  AndroidApi: $AndroidApi"
Write-Host "  JavaPkg   : $JavaPkg"
Write-Host "  OutFile   : $outPath"
Write-Host "  Gomobile  : golang.org/x/mobile@$GomobileVersion"

if (-not (Test-Path $moduleDir)) {
  throw "nodemobile module not found: $moduleDir"
}

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $outPath) | Out-Null

Write-Host "Installing pinned gomobile..." -ForegroundColor Cyan
$goBin = Join-Path (& go env GOPATH) 'bin'
if (-not (($env:Path -split [IO.Path]::PathSeparator) -contains $goBin)) {
  $env:Path = "$goBin$([IO.Path]::PathSeparator)$env:Path"
}
go install "golang.org/x/mobile/cmd/gomobile@$GomobileVersion"
if ($LASTEXITCODE -ne 0) {
  throw "go install gomobile failed (exit=$LASTEXITCODE)"
}
go install "golang.org/x/mobile/cmd/gobind@$GomobileVersion"
if ($LASTEXITCODE -ne 0) {
  throw "go install gobind failed (exit=$LASTEXITCODE)"
}

$env:GOWORK = 'off'

Push-Location $moduleDir
try {
  Write-Host "Running: gomobile init" -ForegroundColor Cyan
  gomobile init
  if ($LASTEXITCODE -ne 0) {
    throw "gomobile init failed (exit=$LASTEXITCODE). Please ensure Android SDK/NDK is installed and ANDROID_HOME is set."
  }

  Write-Host "Running: gomobile bind" -ForegroundColor Cyan
  gomobile bind -target $Target -androidapi $AndroidApi -javapkg $JavaPkg -o $outPath .
  if ($LASTEXITCODE -ne 0) {
    throw "gomobile bind failed (exit=$LASTEXITCODE). Please ensure Android SDK/NDK is installed and ANDROID_HOME is set."
  }
} finally {
  Pop-Location
}

if (-not (Test-Path $outPath)) {
  throw "AAR not generated: $outPath"
}

Write-Host "OK: $outPath" -ForegroundColor Green

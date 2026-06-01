param(
  [string]$Target = 'ios',
  [string]$OutFile = 'app/ios/Frameworks/Nodemobile.xcframework',
  [string]$GomobileVersion = 'v0.0.0-20260217195705-b56b3793a9c4'
)

$ErrorActionPreference = 'Stop'

if (-not $IsMacOS) {
  throw 'iOS gomobile binding requires macOS and Xcode. Run scripts/build_ios_xcframework.sh on macOS.'
}

$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path | Split-Path -Parent
$script = Join-Path $repoRoot 'scripts/build_ios_xcframework.sh'
$env:GOMOBILE_VERSION = $GomobileVersion
& bash $script $Target $OutFile
if ($LASTEXITCODE -ne 0) {
  throw "build_ios_xcframework.sh failed (exit=$LASTEXITCODE)"
}

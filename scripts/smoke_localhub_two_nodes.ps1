param(
  [string]$ServerRoot = "D:\project\MyFlowHub3\repo\MyFlowHub-Server",
  [string]$BridgeExe = "build/clipboardnode-bridge.exe",
  [int]$HubPort = 19090,
  [int]$BridgePortA = 18311,
  [int]$BridgePortB = 18312
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path | Split-Path -Parent
$bridgePath = Join-Path $repoRoot $BridgeExe
if (-not (Test-Path $bridgePath)) {
  throw "bridge executable not found: $bridgePath. Run scripts/validate.ps1 first."
}
if (-not (Test-Path $ServerRoot)) {
  throw "MyFlowHub-Server root not found: $ServerRoot"
}

$smokeRoot = Join-Path $env:TEMP ("clipboardnode-localhub-smoke-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $smokeRoot | Out-Null

$hubDir = Join-Path $smokeRoot "hub"
$nodeADir = Join-Path $smokeRoot "node-a"
$nodeBDir = Join-Path $smokeRoot "node-b"
New-Item -ItemType Directory -Force -Path $hubDir, $nodeADir, $nodeBDir | Out-Null

$hubLog = Join-Path $hubDir "hub.log"
$hubErr = Join-Path $hubDir "hub.err"
$hubEndpoint = "127.0.0.1:$HubPort"
$topic = "codex-local-smoke-" + [guid]::NewGuid().ToString("N")

$hubArgs = @(
  "run", "./cmd/hub_server",
  "--addr", $hubEndpoint,
  "--workdir", $hubDir,
  "--auth-default-role", "node",
  "--auth-default-perms", "*",
  "--auth-role-perms", "node:*;admin:*"
)

$hub = Start-Process -FilePath "go" -ArgumentList $hubArgs -WorkingDirectory $ServerRoot -RedirectStandardOutput $hubLog -RedirectStandardError $hubErr -PassThru -WindowStyle Hidden

try {
  $deadline = (Get-Date).AddSeconds(45)
  do {
    Start-Sleep -Milliseconds 500
    $hubReady = Test-NetConnection -ComputerName 127.0.0.1 -Port $HubPort -InformationLevel Quiet
  } until ($hubReady -or (Get-Date) -gt $deadline -or $hub.HasExited)
  if (-not $hubReady) {
    throw "local hub did not start listening on $hubEndpoint"
  }

  $tokenA = "token-a-" + [guid]::NewGuid().ToString("N")
  $tokenB = "token-b-" + [guid]::NewGuid().ToString("N")
  $baseConfig = @{
    enabled = $true
    parent_endpoint = $hubEndpoint
    topic = $topic
    max_inline_bytes = 65536
    auto_watch = $false
    auto_apply = $false
    history_retention = "none"
  }
  $cfgA = $baseConfig.Clone()
  $cfgA.device_label = "codex-local-a"
  $cfgB = $baseConfig.Clone()
  $cfgB.device_label = "codex-local-b"

  $configA = Join-Path $nodeADir "config.json"
  $configB = Join-Path $nodeBDir "config.json"
  $cfgA | ConvertTo-Json | Set-Content -LiteralPath $configA -Encoding UTF8
  $cfgB | ConvertTo-Json | Set-Content -LiteralPath $configB -Encoding UTF8

  $procA = Start-Process -FilePath $bridgePath -ArgumentList @("--config", $configA, "--web-listen", "127.0.0.1:$BridgePortA", "--web-token", $tokenA) -RedirectStandardOutput (Join-Path $nodeADir "stdout.log") -RedirectStandardError (Join-Path $nodeADir "stderr.log") -PassThru -WindowStyle Hidden
  $procB = Start-Process -FilePath $bridgePath -ArgumentList @("--config", $configB, "--web-listen", "127.0.0.1:$BridgePortB", "--web-token", $tokenB) -RedirectStandardOutput (Join-Path $nodeBDir "stdout.log") -RedirectStandardError (Join-Path $nodeBDir "stderr.log") -PassThru -WindowStyle Hidden

  try {
    Start-Sleep -Seconds 2
    $headersA = @{ "X-ClipboardNode-Token" = $tokenA }
    $headersB = @{ "X-ClipboardNode-Token" = $tokenB }
    $connect = @{ id = "connect"; action = "connect" } | ConvertTo-Json -Compress
    $connectA = Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:$BridgePortA/command" -Headers $headersA -ContentType "application/json" -Body $connect -TimeoutSec 20
    $connectB = Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:$BridgePortB/command" -Headers $headersB -ContentType "application/json" -Body $connect -TimeoutSec 20
    $statusA = Invoke-RestMethod -Uri "http://127.0.0.1:$BridgePortA/status" -Headers $headersA -TimeoutSec 5
    $statusB = Invoke-RestMethod -Uri "http://127.0.0.1:$BridgePortB/status" -Headers $headersB -TimeoutSec 5

    $text = "codex smoke " + [guid]::NewGuid().ToString("N")
    $send = @{ id = "send"; action = "send_text"; data = @{ text = $text } } | ConvertTo-Json -Depth 4 -Compress
    $sendA = Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:$BridgePortA/command" -Headers $headersA -ContentType "application/json" -Body $send -TimeoutSec 20

    $statusBAfter = $null
    for ($i = 0; $i -lt 20; $i++) {
      Start-Sleep -Milliseconds 500
      $statusBAfter = Invoke-RestMethod -Uri "http://127.0.0.1:$BridgePortB/status" -Headers $headersB -TimeoutSec 5
      if ($statusBAfter.pending_event_id -or $statusBAfter.last_action -eq "remote_pending" -or $statusBAfter.last_action -eq "remote_applied") {
        break
      }
    }

    $success = [bool](
      $connectA.ok -and
      $connectB.ok -and
      $statusA.logged_in -and
      $statusB.logged_in -and
      $statusA.subscribed -and
      $statusB.subscribed -and
      $sendA.ok -and
      ($statusBAfter.pending_event_id -or $statusBAfter.last_action -eq "remote_pending" -or $statusBAfter.last_action -eq "remote_applied")
    )

    $result = [ordered]@{
      success = $success
      smoke_dir = $smokeRoot
      hub_endpoint = $hubEndpoint
      topic = $topic
      node_a = @{
        node_id = $statusA.node_id
        logged_in = $statusA.logged_in
        subscribed = $statusA.subscribed
        last_action = $sendA.status.last_action
        last_event_id = $sendA.status.last_event_id
        last_size = $sendA.status.last_size
        last_hash_prefix = $sendA.status.last_hash_prefix
      }
      node_b = @{
        node_id = $statusB.node_id
        logged_in = $statusB.logged_in
        subscribed = $statusB.subscribed
        pending_event_id = $statusBAfter.pending_event_id
        pending_size = $statusBAfter.pending_size
        pending_hash_prefix = $statusBAfter.pending_hash_prefix
        last_action = $statusBAfter.last_action
        last_event_id = $statusBAfter.last_event_id
        last_size = $statusBAfter.last_size
        last_hash_prefix = $statusBAfter.last_hash_prefix
      }
    }
    $result | ConvertTo-Json -Depth 6
    if (-not $success) {
      throw "local two-node smoke did not reach pending/apply state"
    }
  } finally {
    if ($procA -and -not $procA.HasExited) { Stop-Process -Id $procA.Id -Force }
    if ($procB -and -not $procB.HasExited) { Stop-Process -Id $procB.Id -Force }
  }
} finally {
  if ($hub -and -not $hub.HasExited) { Stop-Process -Id $hub.Id -Force }
}

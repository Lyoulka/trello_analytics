# Локальные команды разработки для Windows PowerShell + Docker Desktop.
# Примеры:
#   .\scripts\dev.ps1 up
#   .\scripts\dev.ps1 logs
#   .\scripts\dev.ps1 migrate-up
#   .\scripts\dev.ps1 db-reset

param(
    [Parameter(Position = 0, Mandatory = $true)]
    [ValidateSet(
        "env-init", "config", "up", "up-build", "stop", "down", "ps",
        "logs", "logs-backend", "logs-frontend", "logs-postgres",
        "migrate-up", "migrate-down", "migrate-version",
        "db-reset", "test", "test-backend", "test-frontend", "help"
    )]
    [string]$Command
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

function Ensure-Env {
    if (-not (Test-Path ".env")) {
        Copy-Item ".env.example" ".env"
        Write-Host "Created .env from .env.example"
    }
}

function Invoke-Migrate {
    param([string]$ArgsLine)
    Ensure-Env
    docker compose --profile tools run --rm --entrypoint /bin/sh migrate `
        -c "migrate -path=/migrations -database `"$env:DATABASE_URL`" $ArgsLine"
}

function Load-DotEnv {
    if (-not (Test-Path ".env")) { return }
    Get-Content ".env" | ForEach-Object {
        $line = $_.Trim()
        if ($line -eq "" -or $line.StartsWith("#")) { return }
        $parts = $line -split "=", 2
        if ($parts.Count -eq 2) {
            [System.Environment]::SetEnvironmentVariable($parts[0], $parts[1], "Process")
        }
    }
}

switch ($Command) {
    "help" {
        Write-Host @"
Commands:
  env-init         Copy .env.example -> .env if missing
  config           Validate docker compose config
  up               Start stack (detached)
  up-build         Rebuild images and start
  stop             Stop containers, keep volumes/data
  down             Remove containers/network, keep volumes
  logs             Follow all logs
  migrate-up       Apply DB migrations
  migrate-down     Roll back one migration
  migrate-version  Show migration version
  db-reset         DESTRUCTIVE: delete DB volume, recreate, migrate
  test             Run backend + frontend tests
  test-backend     go test ./...
  test-frontend    npm test
"@
    }
    "env-init" { Ensure-Env }
    "config" { docker compose config }
    "up" {
        Ensure-Env
        docker compose up -d
    }
    "up-build" {
        Ensure-Env
        docker compose up -d --build
    }
    "stop" { docker compose stop }
    "down" { docker compose down }
    "ps" { docker compose ps }
    "logs" { docker compose logs -f --tail=200 }
    "logs-backend" { docker compose logs -f --tail=200 backend }
    "logs-frontend" { docker compose logs -f --tail=200 frontend }
    "logs-postgres" { docker compose logs -f --tail=200 postgres }
    "migrate-up" {
        Load-DotEnv
        Ensure-Env
        docker compose --profile tools run --rm migrate
    }
    "migrate-down" {
        Load-DotEnv
        Ensure-Env
        docker compose --profile tools run --rm --entrypoint /bin/sh migrate `
            -c 'migrate -path=/migrations -database "$DATABASE_URL" down 1'
    }
    "migrate-version" {
        Load-DotEnv
        Ensure-Env
        docker compose --profile tools run --rm --entrypoint /bin/sh migrate `
            -c 'migrate -path=/migrations -database "$DATABASE_URL" version'
    }
    "db-reset" {
        Write-Host "WARNING: This deletes local PostgreSQL volume data."
        Ensure-Env
        docker compose down -v
        docker compose up -d --build
        Load-DotEnv
        docker compose --profile tools run --rm migrate
    }
    "test-backend" { docker compose exec -T backend go test ./... }
    "test-frontend" { docker compose exec -T frontend npm test }
    "test" {
        docker compose exec -T backend go test ./...
        docker compose exec -T frontend npm test
    }
}

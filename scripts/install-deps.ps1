# install-deps.ps1
# Script para instalar todas las dependencias del proyecto

Write-Host "==================================" -ForegroundColor Cyan
Write-Host "  Instalando Dependencias" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""

# Verificar que Go está instalado
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "Error: Go no está instalado o no está en el PATH" -ForegroundColor Red
    Write-Host "Descarga Go desde: https://golang.org/dl/" -ForegroundColor Yellow
    exit 1
}

$goVersion = go version
Write-Host "Go encontrado: $goVersion" -ForegroundColor Green
Write-Host ""

# Lista de dependencias
$dependencies = @(
    "github.com/spf13/cobra@latest",
    "github.com/spf13/viper@latest",
    "github.com/anacrolix/torrent@latest",
    "github.com/sirupsen/logrus@latest",
    "github.com/schollz/progressbar/v3@latest",
    "github.com/fatih/color@latest",
    "gopkg.in/yaml.v3@latest"
)

Write-Host "Instalando dependencias principales..." -ForegroundColor Yellow
Write-Host ""

$total = $dependencies.Count
$current = 0

foreach ($dep in $dependencies) {
    $current++
    $depName = $dep.Split("@")[0]
    Write-Host "[$current/$total] Instalando $depName..." -ForegroundColor Cyan
    
    $result = go get $dep 2>&1
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  OK" -ForegroundColor Green
    } else {
        Write-Host "  ADVERTENCIA: Posible error" -ForegroundColor Yellow
        Write-Host "  $result" -ForegroundColor Gray
    }
}

Write-Host ""
Write-Host "Descargando todas las dependencias..." -ForegroundColor Yellow
go mod download

Write-Host ""
Write-Host "Limpiando dependencias no utilizadas..." -ForegroundColor Yellow
go mod tidy

Write-Host ""
Write-Host "Verificando módulos..." -ForegroundColor Yellow
$verifyResult = go mod verify 2>&1

if ($LASTEXITCODE -eq 0) {
    Write-Host "OK - Todos los módulos verificados" -ForegroundColor Green
} else {
    Write-Host "ADVERTENCIA: $verifyResult" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "==================================" -ForegroundColor Cyan
Write-Host "  Instalación Completada" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Dependencias instaladas:" -ForegroundColor Green
go list -m all | Select-Object -First 10

Write-Host ""
Write-Host "Siguiente paso: Ejecutar '.\build.ps1 build' para compilar" -ForegroundColor Yellow
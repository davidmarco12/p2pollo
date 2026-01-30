# build.ps1 - Script de build para Windows

param(
    [string]$Command = "build"
)

$BINARY_NAME = "streaming-cli.exe"
$VERSION = "0.1.0"
$BUILD_DIR = ".\bin"

function Build {
    Write-Host "Compilando..." -ForegroundColor Green
    New-Item -ItemType Directory -Force -Path $BUILD_DIR | Out-Null
    go build -o "$BUILD_DIR\$BINARY_NAME" -ldflags="-X 'main.Version=$VERSION'" .\cmd\main.go
    if ($LASTEXITCODE -eq 0) {
        Write-Host "OK - Binario creado en $BUILD_DIR\$BINARY_NAME" -ForegroundColor Green
    }
}

function Clean {
    Write-Host "Limpiando..." -ForegroundColor Yellow
    Remove-Item -Recurse -Force $BUILD_DIR -ErrorAction SilentlyContinue
    Remove-Item coverage.txt, coverage.html -ErrorAction SilentlyContinue
    go clean
    Write-Host "OK - Limpieza completada" -ForegroundColor Green
}

function Test {
    Write-Host "Ejecutando tests..." -ForegroundColor Green
    go test -v .\...
}

function Run {
    Write-Host "Ejecutando..." -ForegroundColor Yellow
    go run .\cmd\main.go
}

function Dev {
    Build
    if ($LASTEXITCODE -eq 0) {
        & "$BUILD_DIR\$BINARY_NAME"
    }
}

function Format {
    Write-Host "Formateando código..." -ForegroundColor Green
    gofmt -s -w .
    Write-Host "OK - Código formateado" -ForegroundColor Green
}

function Deps {
    Write-Host "Descargando dependencias..." -ForegroundColor Green
    go mod download
    go mod tidy
    Write-Host "OK - Dependencias actualizadas" -ForegroundColor Green
}

function Install-Deps {
    Write-Host "================================" -ForegroundColor Cyan
    Write-Host " Instalando Dependencias" -ForegroundColor Cyan
    Write-Host "================================" -ForegroundColor Cyan
    Write-Host ""
    
    $deps = @(
        "github.com/spf13/cobra@latest",
        "github.com/spf13/viper@latest",
        "github.com/anacrolix/torrent@latest",
        "github.com/sirupsen/logrus@latest",
        "github.com/schollz/progressbar/v3@latest",
        "github.com/fatih/color@latest",
        "gopkg.in/yaml.v3@latest"
    )
    
    $i = 1
    $total = $deps.Count
    
    foreach ($dep in $deps) {
        $depName = $dep.Split("@")[0]
        Write-Host "[$i/$total] Instalando $depName..." -ForegroundColor Cyan
        go get $dep 2>&1 | Out-Null
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host "      OK" -ForegroundColor Green
        } else {
            Write-Host "      ADVERTENCIA" -ForegroundColor Yellow
        }
        $i++
    }
    
    Write-Host ""
    Deps
    
    Write-Host ""
    Write-Host "================================" -ForegroundColor Cyan
    Write-Host " Instalación Completada" -ForegroundColor Cyan
    Write-Host "================================" -ForegroundColor Cyan
}

function List-Deps {
    Write-Host "Dependencias instaladas:" -ForegroundColor Green
    Write-Host ""
    go list -m all
}

function Verify {
    Write-Host "Verificando módulos..." -ForegroundColor Green
    go mod verify
    if ($LASTEXITCODE -eq 0) {
        Write-Host "OK - Todos los módulos verificados" -ForegroundColor Green
    }
}

function Help {
    Write-Host @"
Comandos disponibles:

  .\build.ps1 build           Compila el binario
  .\build.ps1 clean           Limpia archivos generados
  .\build.ps1 test            Ejecuta tests
  .\build.ps1 run             Ejecuta directamente sin compilar
  .\build.ps1 dev             Compila y ejecuta
  .\build.ps1 fmt             Formatea el código
  .\build.ps1 deps            Descarga dependencias
  .\build.ps1 install-deps    Instala todas las dependencias
  .\build.ps1 list-deps       Lista dependencias instaladas
  .\build.ps1 verify          Verifica módulos
  .\build.ps1 help            Muestra esta ayuda

Ejemplos:
  .\build.ps1 install-deps    # Primera vez
  .\build.ps1 build           # Compilar
  .\build.ps1 dev             # Desarrollar

"@
}

# Ejecutar comando
switch ($Command.ToLower()) {
    "build"        { Build }
    "clean"        { Clean }
    "test"         { Test }
    "run"          { Run }
    "dev"          { Dev }
    "fmt"          { Format }
    "format"       { Format }
    "deps"         { Deps }
    "install-deps" { Install-Deps }
    "list-deps"    { List-Deps }
    "verify"       { Verify }
    "help"         { Help }
    default        { Help }
}
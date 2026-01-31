# Variables
BINARY_NAME=p2pollo
VERSION?=0.1.0
BUILD_DIR=./bin
GO_FILES=$(shell find . -name '*.go' -type f -not -path "./vendor/*")

# Colores para output
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: all build clean test coverage lint fmt vet help install dev run

# Default target
all: clean fmt vet test build

## help: Muestra esta ayuda
help:
	@echo "Comandos disponibles:"
	@echo ""
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## build: Compila el binario
build:
	@echo "$(GREEN)Compilando...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) -ldflags="-X 'main.Version=$(VERSION)'" ./cmd/main.go
	@echo "$(GREEN)✓ Binario creado en $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

## dev: Compila y ejecuta en modo desarrollo
dev: build
	@echo "$(YELLOW)Ejecutando en modo desarrollo...$(NC)"
	@$(BUILD_DIR)/$(BINARY_NAME)

## run: Ejecuta directamente sin compilar
run:
	@go run ./cmd/main.go

## install: Instala el binario en $GOPATH/bin
install:
	@echo "$(GREEN)Instalando...$(NC)"
	@go install -ldflags="-X 'main.Version=$(VERSION)'" ./cmd/main.go
	@echo "$(GREEN)✓ Instalado exitosamente$(NC)"

## clean: Limpia archivos compilados y temporales
clean:
	@echo "$(YELLOW)Limpiando...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.txt coverage.html
	@go clean
	@echo "$(GREEN)✓ Limpieza completada$(NC)"

## test: Ejecuta todos los tests
test:
	@echo "$(GREEN)Ejecutando tests...$(NC)"
	@go test -v ./...

## coverage: Genera reporte de cobertura de tests
coverage:
	@echo "$(GREEN)Generando reporte de cobertura...$(NC)"
	@go test -coverprofile=coverage.txt -covermode=atomic ./...
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "$(GREEN)✓ Reporte generado en coverage.html$(NC)"

## lint: Ejecuta el linter (requiere golangci-lint instalado)
lint:
	@echo "$(GREEN)Ejecutando linter...$(NC)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)golangci-lint no está instalado. Instalando...$(NC)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run ./...; \
	fi

## fmt: Formatea el código
fmt:
	@echo "$(GREEN)Formateando código...$(NC)"
	@gofmt -s -w $(GO_FILES)
	@echo "$(GREEN)✓ Código formateado$(NC)"

## vet: Ejecuta go vet
vet:
	@echo "$(GREEN)Ejecutando go vet...$(NC)"
	@go vet ./...
	@echo "$(GREEN)✓ go vet completado$(NC)"

## deps: Descarga las dependencias
deps:
	@echo "$(GREEN)Descargando dependencias...$(NC)"
	@go mod download
	@go mod tidy
	@echo "$(GREEN)✓ Dependencias actualizadas$(NC)"

## build-all: Compila para múltiples plataformas
build-all:
	@echo "$(GREEN)Compilando para múltiples plataformas...$(NC)"
	@mkdir -p $(BUILD_DIR)
	# Linux
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/main.go
	# macOS
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/main.go
	@GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/main.go
	# Windows
	@GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/main.go
	@echo "$(GREEN)✓ Compilación multi-plataforma completada$(NC)"

## docker-build: Construye imagen Docker
docker-build:
	@echo "$(GREEN)Construyendo imagen Docker...$(NC)"
	@docker build -t $(BINARY_NAME):$(VERSION) .
	@echo "$(GREEN)✓ Imagen Docker creada$(NC)"

## version: Muestra la versión
version:
	@echo "$(BINARY_NAME) version $(VERSION)"
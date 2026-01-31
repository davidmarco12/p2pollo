# 🚀 Ejemplo Práctico de Profiling

## Escenario: Analizar p2pollo mientras descarga un video

### Terminal 1: Iniciar p2pollo con profiling

```powershell
cd d:\Proyectos\p2pollo

# Crear un archivo main-profile.go para agregar pprof
cat > cmd/profile-main.go << 'EOF'
//+build ignore

package main

import (
    "fmt"
    "net/http"
    _ "net/http/pprof"
    "time"
)

func main() {
    // Inicia servidor de profiling
    go func() {
        fmt.Println("🔍 PPROF PROFILE INICIADO")
        fmt.Println("   http://localhost:6060/debug/pprof/")
        fmt.Println("")
        fmt.Println("COMANDOS:")
        fmt.Println("  Heap (memoria):")
        fmt.Println("    go tool pprof http://localhost:6060/debug/pprof/heap")
        fmt.Println("")
        fmt.Println("  CPU (tiempo):")
        fmt.Println("    go tool pprof http://localhost:6060/debug/pprof/profile?seconds=10")
        fmt.Println("")
        fmt.Println("  Goroutines:")
        fmt.Println("    go tool pprof http://localhost:6060/debug/pprof/goroutine")
        fmt.Println("")
        fmt.Println("  Mutex:")
        fmt.Println("    go tool pprof http://localhost:6060/debug/pprof/mutex")
        fmt.Println("")
        
        if err := http.ListenAndServe("localhost:6060", nil); err != nil {
            panic(err)
        }
    }()
    
    // Esperar para que puedas conectarte
    fmt.Println("✓ Servidor de profiling iniciado")
    fmt.Println("  Conecta desde otra terminal en 3 segundos...")
    time.Sleep(3 * time.Second)
    
    // Aquí irían los comandos del programa original
}
EOF
```

### Opción A: Ver Memoria en Vivo

```powershell
# Terminal 1 (ejecuta p2pollo)
.\p2pollo.exe play "magnet:..." --file 5

# Terminal 2 (analiza memoria)
go tool pprof http://localhost:6060/debug/pprof/heap

# En pprof:
(pprof) top10              # Top 10 funciones por memoria
(pprof) list copyData      # Ver código de copyData()
(pprof) web                # Genera gráfico (requiere graphviz)
```

### Opción B: CPU Profile

```powershell
# Terminal 1 (ejecuta p2pollo)
.\p2pollo.exe play "magnet:..." --file 5

# Terminal 2 (profile CPU por 10 segundos)
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=10

# En pprof:
(pprof) top
(pprof) list main
```

### Opción C: Ver Goroutines

```powershell
# Mientras p2pollo corre:
go tool pprof http://localhost:6060/debug/pprof/goroutine

(pprof) top              # Cuáles goroutines consumen más
(pprof) traces           # Ver traces
```

### Opción D: Mutex Contention

```powershell
go tool pprof http://localhost:6060/debug/pprof/mutex

(pprof) top              # Dónde hay contención de locks
```

---

## Profiling Offline (Archivos)

Si prefieres no usar HTTP, puedes generar archivos:

```bash
# CPU Profile
go test -cpuprofile=cpu.prof -bench=. ./...
go tool pprof cpu.prof

# Memory Profile
go test -memprofile=mem.prof -bench=. ./...
go tool pprof mem.prof

# Ambos
go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./...
```

---

## Benchmark Rápido

```bash
# Ver benchmarks disponibles
cd d:\Proyectos\p2pollo\internal\player
go test -bench=. -benchmem -run=^$ ./...

# Generar resultados
go test -bench=. -benchmem > results.txt
cat results.txt
```

---

## Ejemplo Output

```
goos: windows
goarch: amd64
pkg: github.com/davidmarco12/p2pollo/internal/player
BenchmarkPlayerCreation-8     	    10000	    102341 ns/op	   1234 B/op	      12 allocs/op
BenchmarkPlayerState-8        	  1000000	      1432 ns/op	    256 B/op	       2 allocs/op
BenchmarkCheckMPV-8           	    50000	     24231 ns/op	    512 B/op	       4 allocs/op
PASS
```

**Significado:**
- `10000`: Iteraciones realizadas
- `102341 ns/op`: 102 microsegundos por operación
- `1234 B/op`: 1234 bytes asignados por operación
- `12 allocs/op`: 12 allocaciones por operación

---

## Comparar Benchmarks

```bash
# Benchmark 1 (versión A)
go test -bench=. -benchmem > old.txt

# Cambiar código

# Benchmark 2 (versión B)
go test -bench=. -benchmem > new.txt

# Comparar
go install golang.org/x/perf/cmd/benchstat@latest
benchstat old.txt new.txt
```

Output:
```
name                  old time/op    new time/op    delta
PlayerCreation-8        102.3µs ± 5%   85.2µs ± 3%  -16.72%  (mejoró 16%)
```

---

## Troubleshooting

### "No connection could be made"
El servidor HTTP no está corriendo. Agrega:
```go
import _ "net/http/pprof"

func init() {
    go http.ListenAndServe("localhost:6060", nil)
}
```

### "graphviz not found"
Instala graphviz:
```bash
# Windows (Chocolatey)
choco install graphviz

# O descarga desde: https://graphviz.org/download/
```

### Ports en uso
Si 6060 está en uso, usa otro puerto:
```go
http.ListenAndServe("localhost:9999", nil)

# Luego conecta a:
go tool pprof http://localhost:9999/debug/pprof/heap
```

---

## Lo que significa cada métrica

| Métrica | Significado | Bueno | Malo |
|---------|-------------|-------|------|
| ns/op | Tiempo por operación | < 100ns | > 1µs |
| B/op | Bytes por operación | < 100B | > 10KB |
| allocs/op | Allocaciones por op | < 5 | > 100 |

---

## En p2pollo: Qué profilear

1. **CopyData()**: ¿Es el buffer lo suficientemente grande?
2. **BytesCompleted()**: ¿Costo del mutex?
3. **Play()**: ¿Cuánto tarda lanzar MPV?
4. **Monitor()**: ¿Consume mucho CPU?

```bash
# Ejemplo completo
go test -cpuprofile=cpu.prof -memprofile=mem.prof \
        -benchmem -bench=. ./internal/player

# Analizar CPU
go tool pprof cpu.prof

# Analizar Memoria
go tool pprof mem.prof
```

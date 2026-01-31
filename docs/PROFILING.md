# 📊 Profiling y Benchmarking en Go

## ¿Qué es Profiling?

**Profiling** es el proceso de medir qué está haciendo tu programa:
- 💾 ¿Cuánta memoria usa?
- ⏱️ ¿Cuál es la función más lenta?
- 🔗 ¿Cuántas goroutines activas hay?
- 🚨 ¿Dónde hay contención de locks?

---

## Tipos de Profiling

### 1. **CPU Profile** ⏱️
Muestra cuánto tiempo de CPU consume cada función.

```bash
# Generar CPU profile
go test -cpuprofile=cpu.prof -bench=. ./...

# Analizar
go tool pprof cpu.prof

# En pprof:
> top       # Top 10 funciones por CPU
> list main # Ver el código
```

### 2. **Memory Profile** 💾
Muestra qué consume más memoria.

```bash
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

### 3. **Goroutine Profile** 🔄
Muestra goroutines activas.

```bash
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### 4. **Mutex Profile** 🔒
Muestra dónde hay contención de locks.

```bash
go test -mutexprofile=mutex.prof ./...
go tool pprof mutex.prof
```

---

## Profiling en Vivo (HTTP)

### Paso 1: Habilitar en tu programa

```go
import (
    "net/http"
    _ "net/http/pprof"
)

func main() {
    // Iniciar servidor de pprof en background
    go http.ListenAndServe("localhost:6060", nil)
    
    // Tu código aquí
}
```

### Paso 2: Ejecutar el programa

```bash
go run .
# Programa corriendo en background
```

### Paso 3: En otra terminal, ejecutar pprof

```bash
# Ver perfiles disponibles
go tool pprof http://localhost:6060/debug/pprof/

# CPU profile (default 30s)
go tool pprof http://localhost:6060/debug/pprof/profile

# Memory (heap) profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutines
go tool pprof http://localhost:6060/debug/pprof/goroutine

# Mutex contention
go tool pprof http://localhost:6060/debug/pprof/mutex
```

---

## Herramientas de pprof

### Interfaz Interactiva

```bash
go tool pprof cpu.prof

# Comandos disponibles:
(pprof) top        # Top 10 por CPU/memoria
(pprof) top10      # Exactamente 10
(pprof) list main  # Ver código de función 'main'
(pprof) web        # Generar gráfico (requiere graphviz)
(pprof) pdf        # Generar PDF
(pprof) png        # Generar PNG
(pprof) text       # Salida de texto
(pprof) quit       # Salir
```

### Interfaz Web

```bash
go tool pprof -http=:8080 cpu.prof
# Abre navegador: http://localhost:8080
```

---

## Ejemplo Práctico: Profiling de p2pollo

### Paso 1: Agregar profiling a play.go

```go
package cmd

import (
    "net/http"
    _ "net/http/pprof"
)

func runPlay(cmd *cobra.Command, args []string) {
    // Habilitar profiling en vivo
    go func() {
        fmt.Println("🔍 Profiling disponible en http://localhost:6060/debug/pprof/")
        http.ListenAndServe("localhost:6060", nil)
    }()
    
    // Resto del código
    magnetLink := args[0]
    // ...
}
```

### Paso 2: Ejecutar con profiling

```bash
# Terminal 1
go run . play "magnet:..." --file 5

# Terminal 2 (mientras p2pollo está corriendo)
go tool pprof http://localhost:6060/debug/pprof/heap
# o
go tool pprof -http=:8080 http://localhost:6060/debug/pprof/heap
```

### Paso 3: Analizar resultados

```
top20                  # Top 20 funciones por memoria
list goroutine_copia   # Ver qué gasta más en copia
web                    # Ver gráfico
```

---

## Benchmarking vs Profiling

| Característica | Benchmark | Profile |
|---|---|---|
| Cuándo | Medir rendimiento específico | Investigar dónde es lento |
| Herramienta | `testing.B` | `pprof` |
| Salida | ns/op, B/op | Gráficos, flamegraphs |
| Uso | CI/CD, regresión | Debugging, optimización |

### Ejemplo Benchmark

```go
func BenchmarkCopiarArchivo(b *testing.B) {
    // Setup
    file, _ := os.Create("test.tmp")
    defer file.Close()
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        // Código a medir
        io.Copy(file, reader)
    }
}
```

```bash
go test -bench=. -benchmem -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

---

## Interpretando Flamegraphs

Un flamegraph muestra:
- **Eje X**: Cantidad de tiempo/memoria
- **Eje Y**: Stack de llamadas
- **Color**: Random (solo para diferencia)
- **Ancho**: Más ancho = más tiempo/memoria

```
main()
├─ runPlay()
│  ├─ client.New()        [████████░░]  40%
│  ├─ copyData()          [██████████]  50%
│  └─ player.Play()       [██░░░░░░░░]  10%
└─ cleanup()              [░░░░░░░░░░]   0%
```

---

## Memory Leaks: Detectando

### Verificar que memoria se libera

```bash
# Tomar snapshot inicial
curl http://localhost:6060/debug/pprof/heap > heap1.prof

# Dejar corriendo 5 minutos
sleep 300

# Tomar snapshot final
curl http://localhost:6060/debug/pprof/heap > heap2.prof

# Comparar
go tool pprof -base heap1.prof heap2.prof
```

Si memoria sigue creciendo: **memory leak**

### Soluciones comunes

```go
// ❌ LEAK: No cerrar canales
done := make(chan bool)
go func() {
    <-done  // Se bloquea para siempre si nadie envía
}()

// ✓ FIX: Cerrar siempre
close(done)  // Despierta todas las goroutines esperando
```

```go
// ❌ LEAK: Goroutines que nunca terminan
for {
    <-ch  // Si ch nunca se cierra, goroutine nunca muere
}

// ✓ FIX: Usar context
select {
case <-ctx.Done():
    return
case <-ch:
    // procesar
}
```

---

## Optimización Basada en Profiling

### Flujo típico

```
1. MEDIR
   ↓
2. IDENTIFICAR BOTTLENECK
   ↓
3. OPTIMIZAR
   ↓
4. VERIFICAR MEJORA
   ↓
5. REPEAT si hay más bottlenecks
```

### Ejemplo en p2pollo

```bash
# 1. MEDIR: Ejecutar con CPU profile
go run . play "magnet:..." --file 5 &
sleep 2
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=10

# 2. IDENTIFICAR: Ver top10
(pprof) top10
# Output: copyData() consuma 60% CPU

# 3. OPTIMIZAR: Aumentar buffer a 256KB
# En play.go cambiar: buf := make([]byte, 256*1024)

# 4. VERIFICAR: Volver a medir
# Repetir paso 1-2, confirmar mejora
```

---

## Herramientas Útiles

### 1. go-torch (Flamegraph)
```bash
go get -u github.com/uber/go-torch
go-torch http://localhost:6060 --time=10
```

### 2. graphviz (para visualizar pprof)
```bash
# Windows (Chocolatey)
choco install graphviz

# Linux
sudo apt-get install graphviz

# macOS
brew install graphviz
```

### 3. Web UI (built-in)
```bash
go tool pprof -http=:8080 cpu.prof
```

---

## Checklist de Performance

- [ ] CPU profile ejecutado
- [ ] Memory profile ejecutado
- [ ] Goroutine profile revisado
- [ ] Mutex profile analizado
- [ ] Sin memory leaks
- [ ] Benchmarks pasan
- [ ] Latency acceptable (< 100ms)
- [ ] Memory footprint reasonable (< 500MB)

---

## Casos Comunes en p2pollo

### 1. Mucha memoria durante descarga

**Síntoma:** Memoria crece constantemente
**Causa probable:** Buffer muy grande en goroutine de copia
**Solución:**
```go
// Reducir buffer
buf := make([]byte, 64*1024)  // Era 256KB
```

### 2. CPU 100% en monitoreo

**Síntoma:** Una goroutine consume mucho CPU
**Causa probable:** Loop sin sleep
**Solución:**
```go
// Agregar delay
ticker := time.NewTicker(500 * time.Millisecond)  // Reducir frecuencia
```

### 3. Goroutines que no terminan

**Síntoma:** NumGoroutine aumenta constantemente
**Causa probable:** Canales no cerrados o deadlock
**Solución:**
```go
// Asegurar close() cuando termina
defer close(done)
defer close(stopCopy)
```

### 4. Contención de Mutex

**Síntoma:** Mutex profile muestra mucho tiempo esperando locks
**Causa probable:** Lock granular muy grueso
**Solución:**
```go
// Usar RWMutex si solo lectura
type Client struct {
    mu sync.RWMutex  // Era sync.Mutex
}

// En lectura: RLock es más rápido
func (c *Client) GetData() {
    c.mu.RLock()     // Era Lock()
    defer c.mu.RUnlock()
}
```

---

## Automatizar Profiling

```go
// profiling_utils.go
package main

import (
    "flag"
    "log"
    "os"
    "runtime"
    "runtime/pprof"
)

var (
    cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
    memprofile = flag.String("memprofile", "", "write memory profile to file")
)

func startProfiling() {
    if *cpuprofile != "" {
        f, err := os.Create(*cpuprofile)
        if err != nil {
            log.Fatal("Could not create CPU profile: ", err)
        }
        
        if err := pprof.StartCPUProfile(f); err != nil {
            log.Fatal("Could not start CPU profile: ", err)
        }
        
        defer pprof.StopCPUProfile()
    }
}

func stopProfiling() {
    if *memprofile != "" {
        f, err := os.Create(*memprofile)
        if err != nil {
            log.Fatal("Could not create memory profile: ", err)
        }
        defer f.Close()
        
        runtime.GC()
        if err := pprof.WriteHeapProfile(f); err != nil {
            log.Fatal("Could not write memory profile: ", err)
        }
    }
}

// En main():
// defer stopProfiling()
// startProfiling()
```

Uso:
```bash
go run . -cpuprofile=cpu.prof -memprofile=mem.prof
```

---

## Conclusion

El profiling es esencial para:
- ✅ Identificar bottlenecks reales
- ✅ Evitar optimizaciones prematuras
- ✅ Detectar memory leaks
- ✅ Validar mejoras de performance

En p2pollo, sería útil profilear:
- Tiempo de copia (¿qué consume más CPU?)
- Uso de memoria durante descarga
- Goroutines activas (¿alguna leak?)
- Contención de mutex en cliente

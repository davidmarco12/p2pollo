# 🚀 Quick Start: Profiling p2pollo

## El error que viste

```powershell
$ go tool pprof http://localhost:6060/debug/pprof/heap

❌ Fetching profile over HTTP from http://localhost:6060/debug/pprof/heap
❌ dial tcp [::1]:6060: connectex: No connection could be made because the target 
   machine actively refused it.
```

**Significado:** No hay nada escuchando en `localhost:6060`.

---

## Cómo funciona ahora

Agregué un flag `--profile` a tu p2pollo. Ahora puedes:

```powershell
# Terminal 1: Iniciar p2pollo CON profiling
.\bin\p2pollo.exe play "magnet:?xt=urn:btih:..." --profile

# Output:
# 🔍 PROFILING ENABLED
#    Servidor en http://localhost:6060/debug/pprof/
#
#    Comandos:
#    Heap:      go tool pprof http://localhost:6060/debug/pprof/heap
#    CPU (10s): go tool pprof http://localhost:6060/debug/pprof/profile?seconds=10
#    Goroutine: go tool pprof http://localhost:6060/debug/pprof/goroutine
#    Mutex:     go tool pprof http://localhost:6060/debug/pprof/mutex
```

---

## Usar profiling en otra terminal

Mientras p2pollo está corriendo CON `--profile`, abre OTRA terminal:

```powershell
# Terminal 2: Analizar memoria en tiempo real
go tool pprof http://localhost:6060/debug/pprof/heap

# Dentro de pprof:
(pprof) top10              # Ver top 10 funciones por memoria
(pprof) list copyData      # Ver código de copyData()
(pprof) web                # Generar gráfico
(pprof) quit               # Salir
```

---

## Ejemplo práctico paso a paso

### 1️⃣ Abre una Terminal (PowerShell)

```powershell
cd d:\Proyectos\p2pollo
```

### 2️⃣ Ejecuta p2pollo CON profiling

```powershell
# Elige un magnet link (ej: Sintel)
$magnet = "magnet:?xt=urn:btih:08ada5c7a6183aae1e09d831df6748f566f12200&dn=Sintel&..."

.\bin\p2pollo.exe play $magnet --file 0 --profile
```

**Esperarás output como:**
```
🔍 PROFILING ENABLED
   Servidor en http://localhost:6060/debug/pprof/
   ...
```

### 3️⃣ Abre otra Terminal (PowerShell) - SIN CERRAR LA PRIMERA

```powershell
# Terminal 2
cd d:\Proyectos\p2pollo

# Mientras p2pollo descarga/streaming:
go tool pprof http://localhost:6060/debug/pprof/heap
```

**Esto abrirá pprof:**
```
Fetching profile over HTTP from http://localhost:6060/debug/pprof/heap
Type: inuse_bytes
Time: Jan 1 2024 at 10:00:00 UTC (0s since start)
Entering interactive mode (type "help" for commands, "o" for options)

(pprof)
```

### 4️⃣ Dentro de pprof, prueba comandos:

```
(pprof) top              # Top 10 funciones que usan más memoria
(pprof) list main        # Código con allocaciones
(pprof) web              # Genera gráfico de memoria
(pprof) quit             # Salir
```

---

## Explicación de los Profile Types

### 🔴 `/heap` - Memoria Actual
```
go tool pprof http://localhost:6060/debug/pprof/heap
```
- **Qué mide**: Memoria asignada EN ESTE MOMENTO
- **Cuándo usar**: Buscar memory leaks durante streaming
- **Ejemplo en p2pollo**: ¿Cuánto buffer usa copyData()?

**Output típico:**
```
(pprof) top
main_profile.go: Contains File: Read()              250MB
main_profile.go: Contains sync.Mutex Lock()         50MB
...
```

### 🟢 `/profile` - CPU (CPU Time)
```
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=10
```
- **Qué mide**: Dónde gasta tiempo la CPU (durante 10 segundos)
- **Cuándo usar**: Bottlenecks de rendimiento
- **Ejemplo en p2pollo**: ¿Qué función tarda más?

### 🟡 `/goroutine` - Goroutines Activas
```
go tool pprof http://localhost:6060/debug/pprof/goroutine
```
- **Qué mide**: Cuántas goroutines hay y dónde están bloqueadas
- **Cuándo usar**: Detectar goroutine leaks
- **Ejemplo en p2pollo**: ¿Tenemos 3 goroutines o más?

### 🟣 `/mutex` - Lock Contention
```
go tool pprof http://localhost:6060/debug/pprof/mutex
```
- **Qué mide**: Dónde hay contención en locks
- **Cuándo usar**: Parallelism problems
- **Ejemplo en p2pollo**: ¿El Mutex de BytesCompleted() es un bottleneck?

---

## Casos de uso en p2pollo

### Caso 1: ¿Toma demasiada memoria?
```powershell
# Terminal 1
.\bin\p2pollo.exe play $magnet --profile

# Terminal 2
go tool pprof http://localhost:6060/debug/pprof/heap
(pprof) top
```

**Si ves:**
```
250MB  main.copyData          <- CopyData goroutine usa mucha memoria
 50MB  main.BytesCompleted    <- Mutex contention
```

🔍 **Análisis**: El buffer de 64KB debería ser más pequeño o el copyData está acumulando algo.

### Caso 2: CPU alta
```powershell
# Terminal 1
.\bin\p2pollo.exe play $magnet --profile

# Terminal 2 (toma 10 segundos de CPU time)
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=10
(pprof) list Monitorear
```

**Si ves funciones llamadas 100,000 veces**:
🔍 **Análisis**: El ticker de 500ms está siendo demasiado agresivo.

### Caso 3: Goroutine Leak
```powershell
# Terminal 1
.\bin\p2pollo.exe play $magnet --profile

# Terminal 2 (ejecuta varias veces durante streaming)
go tool pprof http://localhost:6060/debug/pprof/goroutine
(pprof) top
```

**Si ves:**
```
10 goroutines (esperabas 3)
```

🔍 **Análisis**: Las goroutines no se están cerrando correctamente al terminar el streaming.

---

## Comandos útiles en pprof

```
(pprof) top10              # Top 10 por uso
(pprof) top20              # Top 20
(pprof) list copyData      # Código de función
(pprof) list main          # Código con allocaciones
(pprof) web                # Genera gráfico (requiere graphviz)
(pprof) pdf                # PDF
(pprof) save               # Guarda perfil
(pprof) help               # Ayuda
(pprof) quit               # Salir
```

---

## Troubleshooting

### ❌ "connectex: No connection could be made"
```
✗ Olvidaste agregar --profile
✓ Solución: .\bin\p2pollo.exe play $magnet --profile
```

### ❌ "address already in use"
El puerto 6060 está en uso. Usa otro:
```powershell
.\bin\p2pollo.exe play $magnet --profile --pprof-port 9999
go tool pprof http://localhost:9999/debug/pprof/heap
```

### ❌ "graphviz not found" (cuando usas `web`)
Instala graphviz:
```powershell
choco install graphviz
# O descarga desde: https://graphviz.org/download/
```

---

## Próximos pasos

1. **Ejecuta** con `--profile` y analiza los 3 casos
2. **Identifica** el mayor consumidor de memoria
3. **Optimiza** esa función (ej: ajusta buffer size)
4. **Compara** antes/después con otro profile

¿Preguntas? 📚 Ver `PROFILING.md` para detalles completos.

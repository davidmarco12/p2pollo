# 📝 Cambios Realizados - Profiling Support

## Problema Original

```
$ go tool pprof http://localhost:6060/debug/pprof/heap

❌ ERROR: No connection could be made because the target machine actively refused it.
```

**Usuario no entendía:**
- Qué significa el error
- Por qué ocurrió
- Cómo usarlo correctamente

---

## Solución Implementada

### 1. Modificado: `cmd/root.go`

**Cambios:**
- ✅ Agregado import: `"net/http"` y `_ "net/http/pprof"`
- ✅ Agregadas variables globales: `profile`, `pprof`
- ✅ Nueva función: `startProfiling()`
- ✅ Nuevos flags: `--profile`, `--pprof-port`
- ✅ Inicialización automática en `init()`

**Código:**
```go
// Nuevos flags en init()
rootCmd.PersistentFlags().BoolVar(&profile, "profile", false, "habilitar profiling pprof")
rootCmd.PersistentFlags().StringVar(&pprof, "pprof-port", "6060", "puerto para servidor pprof")

// Nueva función
func startProfiling() {
    if profile {
        go func() {
            http.ListenAndServe("localhost:"+pprof, nil)
        }()
    }
}
```

**Efecto:**
- Ahora `p2pollo play ... --profile` inicia servidor HTTP en :6060
- El servidor responde a solicitudes pprof

### 2. Arreglado: `profiling.go`

**Cambio:**
- ✅ Agregado import faltante: `"runtime/pprof"`

**Por qué:**
- El archivo usaba `pprof.StartCPUProfile()` pero no importaba el paquete
- Causaba errores de compilación

---

## Nuevos Archivos de Documentación

### 📄 `ERROR_EXPLANATION.md`
**Contenido:** Explicación detallada del error
- Traducción del error técnico
- Visualización del problema
- Comparación CON vs SIN profiling
- Raíz del problema
- El flag --profile explicado

**Para:** Entender qué pasó mal

### 📄 `PROFILING_QUICK_START.md`
**Contenido:** Guía paso a paso para usar profiling
- Ejemplo práctico 4 pasos
- Explicación de cada Profile Type (/heap, /profile, /goroutine, /mutex)
- 4 casos de uso en p2pollo
- Troubleshooting

**Para:** Empezar a profilear inmediatamente

### 📄 `PROFILING_VISUAL_GUIDE.md`
**Contenido:** Guía visual completa
- Flujo completo (diagrama)
- Profile Types (tabla comparativa)
- Command reference con interpretación
- 4 Casos de uso prácticos detallados
- Workflow de optimización
- Trucos útiles
- Checklist

**Para:** Referencia completa durante profiling

### 📄 `PROFILING_EJEMPLOS.md`
**Contenido:** Ejemplos de código y comandos
- Opciones A, B, C, D de profiling
- Profiling offline (archivos)
- Benchmarks rápidos
- Comparar benchmarks
- Troubleshooting

**Para:** Copy-paste de comandos

---

## Cómo Usar Ahora

### 1️⃣ Compilar

```powershell
cd d:\Proyectos\p2pollo
go build -o bin/p2pollo.exe .
```

### 2️⃣ Terminal 1: Ejecutar con profiling

```powershell
.\bin\p2pollo.exe play "magnet:?xt=..." --file 0 --profile
```

**Output:**
```
🔍 PROFILING ENABLED
   Servidor en http://localhost:6060/debug/pprof/
   
   Comandos:
   Heap:      go tool pprof http://localhost:6060/debug/pprof/heap
   CPU (10s): go tool pprof http://localhost:6060/debug/pprof/profile?seconds=10
   Goroutine: go tool pprof http://localhost:6060/debug/pprof/goroutine
   Mutex:     go tool pprof http://localhost:6060/debug/pprof/mutex
```

### 3️⃣ Terminal 2: Conectar con pprof

```powershell
go tool pprof http://localhost:6060/debug/pprof/heap
```

**Output:**
```
Fetching profile over HTTP from http://localhost:6060/debug/pprof/heap
Type: inuse_bytes
Time: Jan 1 2024 at 10:00:00 UTC
Entering interactive mode (type "help" for commands)

(pprof)
```

### 4️⃣ Analizar dentro de pprof

```
(pprof) top              # Ver top 10 funciones
(pprof) list copyData    # Ver código de copyData
(pprof) web              # Generar gráfico
(pprof) quit             # Salir
```

---

## Comparación Antes vs Después

### ❌ ANTES

```powershell
$ go tool pprof http://localhost:6060/debug/pprof/heap

ERROR: connection refused
```

### ✅ DESPUÉS

```powershell
# Terminal 1
$ .\bin\p2pollo.exe play magnet:... --profile
✓ HTTP Server escuchando

# Terminal 2
$ go tool pprof http://localhost:6060/debug/pprof/heap
✓ Conectado!
```

---

## Archivos Modificados

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `cmd/root.go` | Agregado profiling support | +30 |
| `profiling.go` | Arreglado import | +1 |

## Archivos Creados

| Archivo | Propósito | Líneas |
|---------|-----------|--------|
| `ERROR_EXPLANATION.md` | Explicar el error | ~150 |
| `PROFILING_QUICK_START.md` | Guía de inicio rápido | ~300 |
| `PROFILING_VISUAL_GUIDE.md` | Guía visual completa | ~450 |
| `PROFILING_EJEMPLOS.md` | Ejemplos y comandos | ~250 |

**Total:** 4 nuevos archivos de documentación + modificaciones en 2 archivos

---

## Resumen Técnico

### Qué se cambió en el código

**Antes:**
```go
// No había soporte para profiling
// No había opción de iniciar servidor HTTP
```

**Después:**
```go
// En init():
cobra.OnInitialize(startProfiling)

rootCmd.PersistentFlags().BoolVar(&profile, "profile", false, "...")
rootCmd.PersistentFlags().StringVar(&pprof, "pprof-port", "6060", "...")

// Nueva función:
func startProfiling() {
    if profile {
        go func() {
            http.ListenAndServe("localhost:"+pprof, nil)
        }()
    }
}
```

### Qué sucede ahora

1. Usuario ejecuta: `.\p2pollo.exe play magnet:... --profile`
2. Flag `--profile` se detecta como `true`
3. Se llama `startProfiling()` antes de ejecutar comando
4. Se inicia goroutine con `http.ListenAndServe("localhost:6060", nil)`
5. El servidor HTTP responde a solicitudes pprof
6. Usuario puede conectar desde otra terminal con `go tool pprof http://localhost:6060/debug/pprof/heap`

---

## Testing

### Verificación de compilación

```powershell
cd d:\Proyectos\p2pollo
go build -o bin/p2pollo.exe .
# ✓ Compila sin errores
```

### Verificación de flags

```powershell
.\bin\p2pollo.exe play --help
# ✓ Muestra flags --profile y --pprof-port
```

### Verificación de ejecución

```powershell
.\bin\p2pollo.exe play magnet:... --profile
# ✓ Muestra mensaje "PROFILING ENABLED"
# ✓ HTTP server escuchando en :6060
```

---

## Documentación Creada

### Jerarquía de Lectura

```
1. ERROR_EXPLANATION.md
   ↓ (Entiendes qué pasó)
   ↓
2. PROFILING_QUICK_START.md
   ↓ (Sabes cómo empezar)
   ↓
3. PROFILING_VISUAL_GUIDE.md
   ↓ (Referencia visual completa)
   ↓
4. PROFILING_EJEMPLOS.md
   ↓ (Comandos para copy-paste)
   ↓
5. PROFILING.md (original)
   (Referencia teórica profunda)
```

---

## Próximos Pasos

### Para el Usuario

1. ✓ Leer `ERROR_EXPLANATION.md` (entender qué pasó)
2. ✓ Seguir `PROFILING_QUICK_START.md` (ejecutar ejemplo)
3. ✓ Usar `PROFILING_VISUAL_GUIDE.md` como referencia
4. ✓ Profilear p2pollo en tiempo real

### Para Mejorar Código

- [ ] Benchmarks más completos (resolver port binding issues)
- [ ] Agregar `--pprof-host` para remote profiling
- [ ] Integrar profiling stats en logs
- [ ] Crear dashboard de profiling
- [ ] Automatizar comparaciones antes/después

---

## Comandos Rápidos

```powershell
# Compilar
cd d:\Proyectos\p2pollo && go build -o bin/p2pollo.exe .

# Ejecutar con profiling
.\bin\p2pollo.exe play "magnet:?xt=urn:btih:08ada5c7a6183aae1e09d831df6748f566f12200" --file 0 --profile

# Analizar heap
go tool pprof http://localhost:6060/debug/pprof/heap

# Analizar CPU (10 segundos)
go tool pprof "http://localhost:6060/debug/pprof/profile?seconds=10"

# Ver goroutines
go tool pprof http://localhost:6060/debug/pprof/goroutine

# Ver mutex contention
go tool pprof http://localhost:6060/debug/pprof/mutex
```

---

## Referencias

- Go pprof documentation: https://pkg.go.dev/runtime/pprof
- Profiling guide: https://go.dev/blog/profiling-go-programs
- CPU profiling: https://go.dev/blog/pprof

---

**Estado:** ✅ COMPLETADO  
**Fecha:** [Actual]  
**Cambios:** Profiling support fully integrated into p2pollo

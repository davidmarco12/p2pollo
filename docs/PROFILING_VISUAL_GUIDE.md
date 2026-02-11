# 🎯 Profiling: Guía Visual Interactiva

## El Flujo Completo (Visual)

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│ 1️⃣  DECIDE qué profilear                                      │
│                                                                 │
│     • Memoria: heap, alloc                                     │
│     • CPU: profile                                             │
│     • Concurrencia: goroutine, mutex                           │
│                                                                 │
└────────────────┬────────────────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│ 2️⃣  INICIA p2pollo CON --profile                             │
│                                                                 │
│ Terminal 1:                                                     │
│ $ .\bin\p2pollo.exe play magnet:... --profile                 │
│   ⏳ Descargando...                                            │
│   🔍 HTTP Server: localhost:6060                              │
│                                                                 │
└────────────────┬────────────────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│ 3️⃣  CONECTA con pprof                                         │
│                                                                 │
│ Terminal 2:                                                     │
│ $ go tool pprof http://localhost:6060/debug/pprof/heap       │
│   ✓ Conectado                                                  │
│   (pprof)                                                       │
│                                                                 │
└────────────────┬────────────────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│ 4️⃣  ANALIZA resultados                                        │
│                                                                 │
│ (pprof) top                 ← Ver funciones por memoria       │
│ (pprof) list copyData       ← Ver código específico           │
│ (pprof) web                 ← Generar gráfico                 │
│ (pprof) quit                ← Salir                            │
│                                                                 │
└────────────────┬────────────────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│ 5️⃣  OPTIMIZA & REPITE                                         │
│                                                                 │
│ Cambio → Recompila → Ejecuta de nuevo con --profile          │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Profile Types (Comparación)

```
┌──────────────────────┬─────────────────────┬──────────────────┐
│ Profile Type         │ Qué Mide            │ Cuándo Usar      │
├──────────────────────┼─────────────────────┼──────────────────┤
│ heap                 │ Memoria ahora       │ Memory leaks      │
│ alloc                │ Total allocado      │ Allocations      │
│ cpu (profile)        │ Tiempo de CPU       │ Slow functions   │
│ goroutine            │ Goroutines activas  │ Goroutine leaks  │
│ mutex                │ Lock contention     │ Parallelism bugs │
│ threadcreate         │ Thread creation     │ System limits    │
└──────────────────────┴─────────────────────┴──────────────────┘
```

---

## Comando Quick Reference

### 🔴 Memoria (Heap)

```powershell
# Mientras p2pollo corre con --profile:
go tool pprof http://localhost:6060/debug/pprof/heap

# Dentro de pprof:
(pprof) top
(pprof) top10 --cum      # Por función cumulativa
(pprof) list copyData    # Ver código de copyData
```

**Interpretación:**

```
Showing nodes accounting for 256MB, 85% of 300MB total

  75MB  25% 25%    75MB  25%  main.copyData
  50MB  17% 42%    50MB  17%  sync.(*Mutex).Lock
  40MB  13% 55%    40MB  13%  main.BytesCompleted
  ...
```

- **75MB**: Memoria usada por copyData
- **25%**: Porcentaje del total
- **25%**: Acumulativo
- **75MB**: Asignaciones propias de copyData

### 🟢 CPU (Time)

```powershell
# Mientras p2pollo corre:
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=10

# Adentro de pprof:
(pprof) top
(pprof) list main
```

**Interpretación:**

```
Showing nodes accounting for 8.50s, 85% of 10s total

  5.00s  50% 50%     5.00s  50%  main.copyData
  2.00s  20% 70%     2.00s  20%  main.Monitorear
  1.50s  15% 85%     1.50s  15%  runtime.mallocgc
```

- **5.00s**: Tiempo en copyData
- **50%**: Porcentaje del tiempo total
- **Esto significa**: 50% del CPU tiempo se gastó en copyData

### 🟡 Goroutines

```powershell
go tool pprof http://localhost:6060/debug/pprof/goroutine

# Adentro:
(pprof) top
```

**Interpretación:**

```
Showing nodes accounting for 5 goroutines

  2 goroutines  40% 40%       2  main.Monitorear
  1 goroutines  20% 60%       1  main.copyData
  1 goroutines  20% 80%       1  runtime.main
  1 goroutines  20% 100%      1  runtime.MHeap_Scavenger
```

- **5 goroutines**: Total de goroutines
- **2 goroutines en Monitorear**: Hay 2 goroutines bloqueadas aquí
- **Esperado**: 3 (copyData, Monitorear buffer, Monitorear download)

### 🟣 Mutex Contention

```powershell
go tool pprof http://localhost:6060/debug/pprof/mutex

# Adentro:
(pprof) top
```

**Interpretación:**

```
Showing nodes accounting for 100ms blocking time

  80ms  80% 80%     80ms  80%  sync.(*Mutex).Lock
  15ms  15% 95%     15ms  15%  main.BytesCompleted
   5ms   5% 100%     5ms   5%  main.Play
```

- **80ms**: El mutex total bloquea 80ms
- **En BytesCompleted**: Hay contención aquí
- **Significa**: Muchas goroutines esperando el mutex

---

## Casos de Uso Prácticos en p2pollo

### 📊 Caso 1: Analizar Memoria

**Pregunta:** ¿Cuánta memoria usa cada parte?

```powershell
# Terminal 1
.\bin\p2pollo.exe play magnet:... --profile

# Terminal 2
go tool pprof http://localhost:6060/debug/pprof/heap

(pprof) top20
(pprof) list copyData    # Ver código con asignaciones
(pprof) web              # Generar gráfico
```

**Árbol esperado:**

```
copyData
├─ buf := make([]byte, 64*1024)     ← 64KB
├─ reader.Read(buf)                 ← otra 64KB
└─ tempFile.Write(buf)              ← otra 64KB
   = ~192KB por iteración

Si copyData itera 1000 veces:
   1000 × 192KB = 192MB
```

**Acción:** Si ves > 500MB, el buffer es demasiado grande.

### ⚡ Caso 2: Encontrar Bottleneck CPU

**Pregunta:** ¿Cuál función es lenta?

```powershell
# Terminal 1
.\bin\p2pollo.exe play magnet:... --profile

# Terminal 2 (perfila CPU durante 10 segundos)
go tool pprof "http://localhost:6060/debug/pprof/profile?seconds=10"

(pprof) top10
(pprof) list Monitorear   # Ver función lenta
```

**Árbol esperado:**

```
  CPU Time    Function
  ────────────────────
  50%         copyData (I/O wait, es normal)
  30%         Monitorear (ticker loop)
  15%         runtime.gc
   5%         autres
```

**Acción:** Si Monitorear > 20%, reducir frecuencia de ticker.

### 🔀 Caso 3: Goroutine Leaks

**Pregunta:** ¿Se cierran correctamente las goroutines?

```powershell
# Ejecuta streaming hasta el final
.\bin\p2pollo.exe play magnet:... --profile

# Cuando termine, rápidamente:
go tool pprof http://localhost:6060/debug/pprof/goroutine

(pprof) top
```

**Escenario 1 - CORRECTO:**

```
5 goroutines
├─ 1 runtime.main
├─ 1 runtime.scavenger
├─ 1 runtime.logwriter
└─ 1 otros (cleaning up)
= Esperado: ~5
```

**Escenario 2 - LEAK:**

```
1000 goroutines!
├─ 900 main.copyData
├─ 95 sync.(*Cond).Wait  
└─ 5 otros
= BAD: Goroutines no se cierran
```

**Acción:** Revisar que copyData() recibe señal de stop.

### 🔐 Caso 4: Lock Contention

**Pregunta:** ¿El Mutex es un bottleneck?

```powershell
# Terminal 1
.\bin\p2pollo.exe play magnet:... --profile

# Terminal 2
go tool pprof http://localhost:6060/debug/pprof/mutex

(pprof) top
```

**Escenario 1 - BIEN:**

```
Total blocking time: 5ms
├─ BytesCompleted Mutex: 3ms (normal, no es mucho)
└─ otros: 2ms
= Parallelism está bien
```

**Escenario 2 - PROBLEMA:**

```
Total blocking time: 5000ms!
├─ BytesCompleted Mutex: 4900ms (MUTEX IS BOTTLENECK)
└─ otros: 100ms
= Todas las goroutines esperando el mutex
```

**Acción:** 
- Cambiar de sync.Mutex a sync.RWMutex (readers)
- Usar atomic.Load() en lugar de mutex si es solo lectura
- Reducir tiempo que el lock está adquirido

---

## Workflow de Optimización

```
START
  ↓
[1] Profile con --profile
  ↓
[2] ¿Hay memory leak?
  ├─ SÍ → heap profile → arregla allocations → goto [1]
  └─ NO ↓
[3] ¿CPU es alta?
  ├─ SÍ → cpu profile → optimiza hot loops → goto [1]
  └─ NO ↓
[4] ¿Goroutine leak?
  ├─ SÍ → goroutine profile → arregla cleanup → goto [1]
  └─ NO ↓
[5] ¿Lock contention?
  ├─ SÍ → mutex profile → arregla sync → goto [1]
  └─ NO ↓
[6] ✅ OPTIMIZADO
```

---

## Trucos Útiles

### Generar Gráfico (SVG)

```powershell
# Instalar graphviz primero
choco install graphviz

# Luego en pprof:
(pprof) web                    # Abre en navegador
(pprof) svg                    # Genera SVG
(pprof) pdf                    # Genera PDF
```

### Comparar Dos Profiles

```powershell
# Profile 1 (antes)
go test -cpuprofile=old.prof -bench=. ./...

# Profile 2 (después)
go test -cpuprofile=new.prof -bench=. ./...

# Comparar
go tool pprof -base=old.prof new.prof
```

### Guardar Profile a Archivo

```powershell
go tool pprof http://localhost:6060/debug/pprof/heap heap.prof

# Luego analizar offline
go tool pprof heap.prof
```

---

## Checklist de Profiling

- [ ] ¿Compilaste con `go build -o bin/p2pollo.exe`?
- [ ] ¿Ejecutaste con `--profile`?
- [ ] ¿Abriste pprof en OTRA terminal?
- [ ] ¿Dejaste suficiente tiempo antes de conectar?
- [ ] ¿Cerraste pprof correctamente?

---

## Comandos Copy-Paste

```powershell
# Setup
cd d:\Proyectos\p2pollo
go build -o bin/p2pollo.exe .

# Terminal 1
.\bin\p2pollo.exe play "magnet:?xt=urn:btih:..." --file 0 --profile

# Terminal 2 (después de 2 segundos)
go tool pprof http://localhost:6060/debug/pprof/heap
```

¡Listo para profilear! 🎉

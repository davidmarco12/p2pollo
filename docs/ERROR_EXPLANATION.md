# 🔴 Error Explicado: "No connection could be made because the target machine actively refused it"

## El Error Exacto que Viste

```
go tool pprof http://localhost:6060/debug/pprof/heap

Fetching profile over HTTP from http://localhost:6060/debug/pprof/heap
http://localhost:6060/debug/pprof/heap: Get "http://localhost:6060/debug/pprof/heap": 
dial tcp [::1]:6060: connectex: No connection could be made because the target machine 
actively refused it.
failed to fetch any source profiles
```

---

## Traducción Simple

| Técnico | Significado |
|---------|-------------|
| `dial tcp [::1]:6060` | Intenta conectar a IPv6 localhost, puerto 6060 |
| `connectex` | Error de conexión (específico de Windows) |
| `connection refused` | **Nadie está escuchando en ese puerto** |

---

## Por Qué Pasó

**Visualización del problema:**

```
┌─────────────────────────────────────────────────────────────┐
│ TERMINAL 1: p2pollo corriendo SIN --profile                 │
├─────────────────────────────────────────────────────────────┤
│ $ .\bin\p2pollo.exe play "magnet:..."                       │
│ ⏳ Descargando... sin servidor de profiling                 │
│                                                              │
│ ❌ NO HAY HTTP SERVER EN PUERTO 6060                        │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ TERMINAL 2: Intentas conectar                               │
├─────────────────────────────────────────────────────────────┤
│ $ go tool pprof http://localhost:6060/debug/pprof/heap      │
│                                                              │
│ 🔍 "Conectaré a localhost:6060..."                          │
│ ❌ "Nadie está escuchando allí"                             │
│ ❌ "Connection refused!"                                    │
└─────────────────────────────────────────────────────────────┘
```

---

## Comparación: CON y SIN Profiling

### ❌ SIN profiling (Lo que pasó)

```powershell
TERMINAL 1:
$ .\bin\p2pollo.exe play "magnet:..."
⏳ Descargando...
❌ Sin servidor HTTP

TERMINAL 2:
$ go tool pprof http://localhost:6060/debug/pprof/heap
❌ ERROR: Connection refused
```

### ✅ CON profiling (Debe ser así)

```powershell
TERMINAL 1:
$ .\bin\p2pollo.exe play "magnet:..." --profile
🔍 PROFILING ENABLED
   Servidor en http://localhost:6060/debug/pprof/
⏳ Descargando...
✓ Servidor HTTP escuchando

TERMINAL 2:
$ go tool pprof http://localhost:6060/debug/pprof/heap
✓ Conectado!
Fetching profile...
Type: inuse_bytes
```

---

## Raíz del Problema: No Agregaste --profile

En tu primer intento ejecutaste:

```powershell
# ❌ INCORRECTO: Sin --profile
go tool pprof http://localhost:6060/debug/pprof/heap
```

Esto es como intentar conectar a un servidor web que no existe.

**La solución requiere DOS PASOS:**

### Paso 1: Iniciar p2pollo CON servidor profiling

```powershell
.\bin\p2pollo.exe play "magnet:..." --profile
```

Este comando:
- Inicia p2pollo normalmente
- **ADEMÁS** inicia un HTTP server en puerto 6060
- Responde a requests de pprof

### Paso 2: Conectar desde otra terminal

```powershell
go tool pprof http://localhost:6060/debug/pprof/heap
```

Ahora:
- Conecta al servidor que está corriendo
- ✓ NO falla con "connection refused"
- ✓ Descarga el perfil de memoria

---

## El Flag --profile

Que agregué hoy:

```go
// En cmd/root.go
rootCmd.PersistentFlags().BoolVar(&profile, "profile", false, "habilitar profiling pprof")
rootCmd.PersistentFlags().StringVar(&pprof, "pprof-port", "6060", "puerto para servidor pprof")

// En startProfiling():
if profile {
    go func() {
        http.ListenAndServe("localhost:" + pprof, nil)
    }()
}
```

**Efecto:**
- `--profile` sin el flag → NO inicia servidor (ahorra recursos)
- `--profile` CON el flag → Inicia servidor HTTP en puerto especificado

---

## Analogía Útil

Es como un restaurante:

```
Escenario 1 (Tu error):
┌─────────────────────────┐
│ CLIENTE (Terminal 2)    │
│ "Quiero pedir comida"   │ → Llama al puerto 6060
└─────────────────────────┘
        ❌ No hay restaurante abierto
        
Escenario 2 (Correcto):
┌──────────────────────┐           ┌──────────────────────┐
│ RESTAURANTE (p2pollo)│           │ CLIENTE (Terminal 2) │
│ ABIERTO en puerto 6060│ ←------- │ Pide información     │
│ Tengo el menú        │           │ Recibe respuesta ✓   │
└──────────────────────┘           └──────────────────────┘
   (--profile)
```

---

## Resumen Rápido

| Concepto | Explicación |
|----------|-------------|
| **El Error** | Intentaste conectar a un servidor que no existía |
| **La Causa** | No agregaste el flag `--profile` a p2pollo |
| **La Solución** | 1) Terminal 1: `--profile` 2) Terminal 2: pprof |
| **El Resultado** | Ahora puedes profilear p2pollo en tiempo real |

---

## Próximo Paso

Lee: `PROFILING_QUICK_START.md` para ver ejemplos prácticos.

O ejecuta directamente:

```powershell
# Terminal 1
.\bin\p2pollo.exe play "magnet:?xt=..." --file 0 --profile

# Terminal 2 (mientras descarga)
go tool pprof http://localhost:6060/debug/pprof/heap
```

¡Y verás que funciona! 🎉

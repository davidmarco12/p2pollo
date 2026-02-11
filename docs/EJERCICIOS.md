# 💻 Ejemplos Prácticos - Go Concurrencia

## Ejercicios Paso a Paso

---

## Ejercicio 1: Mi Primer Canal

### Problema
Tienes 2 goroutines que necesitan comunicarse. Una produce números, otra los consume.

### Solución Básica

```go
package main

import "fmt"

func main() {
    // 1. Crear canal de enteros
    numbers := make(chan int)
    
    // 2. Goroutine PRODUCTORA
    go func() {
        for i := 1; i <= 5; i++ {
            fmt.Println("📤 Enviando:", i)
            numbers <- i  // Envía al canal
        }
        close(numbers)  // Señala que terminó
    }()
    
    // 3. Goroutine CONSUMIDORA
    go func() {
        for num := range numbers {  // Recibe del canal
            fmt.Println("📥 Recibí:", num)
        }
        fmt.Println("✓ Consumidor terminó")
    }()
    
    // 4. Esperar a que ambas terminen
    time.Sleep(1 * time.Second)
}
```

**Salida esperada:**
```
📤 Enviando: 1
📥 Recibí: 1
📤 Enviando: 2
📥 Recibí: 2
📤 Enviando: 3
📥 Recibí: 3
📤 Enviando: 4
📥 Recibí: 4
📤 Enviando: 5
📥 Recibí: 5
✓ Consumidor terminó
```

### Conceptos Clave

- `ch <- value` : **ENVIAR** (bloqueante si no hay receptor)
- `value := <-ch` : **RECIBIR** (bloqueante si no hay datos)
- `for value := range ch` : **RECIBIR EN LOOP** (termina cuando close)
- `close(ch)` : **CERRAR CANAL** (todo receptor recibe y sale del loop)

---

## Ejercicio 2: Canales con Buffer

### Problema
Los productores son muy rápidos y los consumidores lentos. Queremos que no se bloqueen tanto.

### Solución

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    // SIN BUFFER: se bloquean mutuamente
    // ch := make(chan int)
    
    // CON BUFFER: permite 3 envíos sin bloquear
    ch := make(chan int, 3)
    
    // Productor (rápido)
    go func() {
        for i := 1; i <= 5; i++ {
            fmt.Printf("📤 Enviando %d (buffer: %d/%d)\n", 
                i, len(ch), cap(ch))
            ch <- i
        }
        close(ch)
    }()
    
    // Consumidor (lento)
    go func() {
        for {
            num, ok := <-ch
            if !ok {
                break
            }
            fmt.Printf("📥 Recibí %d, esperando...\n", num)
            time.Sleep(1 * time.Second)
        }
    }()
    
    time.Sleep(10 * time.Second)
}
```

**Visualización del Buffer:**

```
Iteración 1:
├─ Envía 1: [1 _ _]  (buffer: 1/3)
├─ Envía 2: [1 2 _]  (buffer: 2/3)
├─ Envía 3: [1 2 3]  (buffer: 3/3)
└─ Envía 4: BLOQUEA (esperando que alguien reciba)

Consumidor recibe 1:
├─ Buffer: [2 3 _]   (buffer: 2/3)
├─ Productor continúa: Envía 4: [2 3 4]
└─ etc.
```

### Conceptos

- **Sin Buffer** (`make(chan int)`): Solo envía si hay receptor listo
- **Con Buffer** (`make(chan int, 10)`): Permite 10 valores sin receptor
- `len(ch)` : Cuántos valores hay en el buffer
- `cap(ch)` : Capacidad total del buffer

---

## Ejercicio 3: Select (La Estrella de Go)

### Problema
Quieres esperar múltiples canales y reaccionar al primero que está listo.

### Solución

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    email := make(chan string)
    notification := make(chan string)
    timeout := time.After(3 * time.Second)
    
    // Simulador de email (llega en 2 segundos)
    go func() {
        time.Sleep(2 * time.Second)
        email <- "📧 Tienes un email nuevo"
    }()
    
    // Simulador de notificación (llega en 5 segundos)
    go func() {
        time.Sleep(5 * time.Second)
        notification <- "🔔 Notificación"
    }()
    
    // Select espera al PRIMERO que esté listo
    select {
    case msg := <-email:
        fmt.Println(msg)
        
    case msg := <-notification:
        fmt.Println(msg)
        
    case <-timeout:
        fmt.Println("⏰ Timeout! Nadie respondió en 3 segundos")
    }
}
```

**Salida esperada:**
```
📧 Tienes un email nuevo
```

**¿Por qué?** 
- Email llega en 2s (✓ antes del timeout)
- Notification llega en 5s (✗ después del timeout)
- Select ejecuta el primero que se gatilla: email

### Variación: Múltiples Selects

```go
// Atender múltiples eventos en loop
for {
    select {
    case msg := <-email:
        fmt.Println("📧", msg)
        
    case msg := <-notification:
        fmt.Println("🔔", msg)
        
    case <-timeout:
        fmt.Println("Timeout!")
        return
        
    default:
        fmt.Println("Esperando...")
        time.Sleep(500 * time.Millisecond)
    }
}
```

---

## Ejercicio 4: Sincronización con WaitGroup

### Problema
Tienes N goroutines y necesitas esperar a que TODAS terminen antes de continuar.

### Solución

```go
package main

import (
    "fmt"
    "sync"
)

func main() {
    var wg sync.WaitGroup
    
    // Decir: "Voy a esperar 3 goroutines"
    wg.Add(3)
    
    // Goroutine 1
    go func() {
        defer wg.Done()  // ✓ Esta goroutine terminó
        fmt.Println("👷 Goroutine 1 trabajando...")
    }()
    
    // Goroutine 2
    go func() {
        defer wg.Done()  // ✓ Esta goroutine terminó
        fmt.Println("👷 Goroutine 2 trabajando...")
    }()
    
    // Goroutine 3
    go func() {
        defer wg.Done()  // ✓ Esta goroutine terminó
        fmt.Println("👷 Goroutine 3 trabajando...")
    }()
    
    // Esperar a que las 3 terminen
    fmt.Println("⏳ Esperando a que terminen...")
    wg.Wait()  // BLOQUEA hasta que wg.Done() se llame 3 veces
    
    fmt.Println("✓ ¡Todas terminaron!")
}
```

**Salida esperada:**
```
⏳ Esperando a que terminen...
👷 Goroutine 1 trabajando...
👷 Goroutine 2 trabajando...
👷 Goroutine 3 trabajando...
✓ ¡Todas terminaron!
```

### Conceptos

- `wg.Add(n)` : Esperar n goroutines
- `wg.Done()` : Decrementar el contador (cuando termina una)
- `defer wg.Done()` : Asegurar que se llama incluso si hay error
- `wg.Wait()` : Bloquea hasta que contador = 0

---

## Ejercicio 5: Patrón Worker Pool (como p2pollo)

### Problema
Tienes 1000 tareas pero solo puedes ejecutar 10 en paralelo.

### Solución

```go
package main

import (
    "fmt"
    "sync"
)

func main() {
    // Canal de tareas
    tareas := make(chan int, 100)
    
    // Canal para señalizar que terminamos
    done := make(chan bool)
    
    // Número de workers
    numWorkers := 3
    var wg sync.WaitGroup
    
    // Lanzar N workers
    for i := 1; i <= numWorkers; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            for tarea := range tareas {  // Recibe tareas
                fmt.Printf("👷 Worker %d: procesando tarea %d\n", id, tarea)
                // Simular trabajo
                time.Sleep(500 * time.Millisecond)
            }
            fmt.Printf("✓ Worker %d terminó\n", id)
        }(i)
    }
    
    // Llenar canal con tareas
    go func() {
        for i := 1; i <= 10; i++ {
            fmt.Printf("📤 Agregando tarea %d\n", i)
            tareas <- i
        }
        close(tareas)  // Señaliza fin de tareas
    }()
    
    // Esperar a que todos los workers terminen
    wg.Wait()
    done <- true
}
```

**Salida esperada:**
```
📤 Agregando tarea 1
📤 Agregando tarea 2
📤 Agregando tarea 3
👷 Worker 1: procesando tarea 1
👷 Worker 2: procesando tarea 2
👷 Worker 3: procesando tarea 3
👷 Worker 1: procesando tarea 4
👷 Worker 2: procesando tarea 5
...
✓ Worker 1 terminó
✓ Worker 2 terminó
✓ Worker 3 terminó
```

---

## Ejercicio 6: Ctx (Context) - El Patrón Moderno

### Problema
Necesitas cancelar múltiples goroutines de forma elegante y propagable.

### Solución

```go
package main

import (
    "context"
    "fmt"
    "time"
)

func worker(ctx context.Context, id int) {
    for {
        select {
        case <-ctx.Done():  // Cancelación
            fmt.Printf("❌ Worker %d cancelado\n", id)
            return
            
        case <-time.After(1 * time.Second):
            fmt.Printf("👷 Worker %d trabajando\n", id)
        }
    }
}

func main() {
    // Crear contexto con cancelación
    ctx, cancel := context.WithCancel(context.Background())
    
    // Lanzar workers
    for i := 1; i <= 3; i++ {
        go worker(ctx, i)
    }
    
    // Dejar trabajar 3 segundos
    time.Sleep(3 * time.Second)
    
    // Cancelar TODO (todas las goroutines recibirán la señal)
    fmt.Println("🛑 Cancelando...")
    cancel()
    
    // Dar tiempo para que salgan
    time.Sleep(1 * time.Second)
    fmt.Println("✓ Hecho")
}
```

**Ventajas de Context:**
- ✓ Cancela automáticamente a todos
- ✓ Propagar deadline a funciones internas
- ✓ Pasar valores entre goroutines
- ✓ Estándar en Go

---

## Ejercicio 7: Mutex - Proteger Datos Compartidos

### Problema
Múltiples goroutines modifican la misma variable = race condition.

### Solución MALA ❌

```go
var counter int  // ¡SIN PROTECCIÓN!

go func() { counter++ }()
go func() { counter++ }()

// Resultado impredecible: 0, 1, o 2
```

### Solución BUENA ✓

```go
var mu sync.Mutex
var counter int

go func() {
    mu.Lock()
    defer mu.Unlock()
    counter++
}()

go func() {
    mu.Lock()
    defer mu.Unlock()
    counter++
}()

// Resultado siempre: 2
```

### Solución CON RWMutex (múltiples lectores)

```go
var mu sync.RWMutex
var data string

// LECTORES (múltiples pueden leer en paralelo)
go func() {
    mu.RLock()
    defer mu.RUnlock()
    fmt.Println("Leyendo:", data)
}()

go func() {
    mu.RLock()
    defer mu.RUnlock()
    fmt.Println("Leyendo:", data)
}()

// ESCRITOR (solo uno por vez)
go func() {
    mu.Lock()
    defer mu.Unlock()
    data = "nuevo valor"
}()
```

---

## Ejercicio 8: Channel Direction (Send/Receive only)

### Problema
Quieres que una función SOLO envíe y otra SOLO reciba (type safety).

### Solución

```go
// Función que SOLO ENVÍA
func producer(ch chan<- int) {
    for i := 1; i <= 5; i++ {
        ch <- i  // OK
        // val := <-ch  // ERROR: invalid operation
    }
}

// Función que SOLO RECIBE
func consumer(ch <-chan int) {
    for val := range ch {
        fmt.Println(val)
        // ch <- 1  // ERROR: invalid operation
    }
}

func main() {
    ch := make(chan int)
    
    go producer(ch)
    go consumer(ch)
    
    time.Sleep(1 * time.Second)
}
```

**Ventajas:**
- Compiler chequea tipos (no envíes donde solo reciben)
- Documentación clara del intent
- Previene bugs

---

## Ejercicio 9: Fan-Out / Fan-In (Como p2pollo)

### Problema
Tienes 1 tarea que necesita procesarse por múltiples workers en paralelo.

### Fan-Out (1 tarea → N workers)

```go
func distribute(nums []int) []<-chan int {
    channels := make([]<-chan int, len(nums))
    
    for i, num := range nums {
        ch := make(chan int)
        channels[i] = ch
        
        go func(n int, c chan int) {
            c <- n * n  // Procesar
            close(c)
        }(num, ch)
    }
    
    return channels
}

// Cada número se procesa en su propia goroutine
results := distribute([]int{1, 2, 3, 4, 5})
```

### Fan-In (N canales → 1 canal)

```go
func merge(channels []<-chan int) <-chan int {
    out := make(chan int)
    var wg sync.WaitGroup
    
    for ch := range channels {
        wg.Add(1)
        go func(c <-chan int) {
            defer wg.Done()
            for val := range c {
                out <- val
            }
        }(ch)
    }
    
    go func() {
        wg.Wait()
        close(out)
    }()
    
    return out
}

// Combinar múltiples canales en uno
results := merge(channels)
for val := range results {
    fmt.Println(val)
}
```

---

## Ejercicio 10: Timeout Pattern (Como en play.go)

### Problema
Una operación puede tardar demasiado. Necesitas un timeout.

### Solución

```go
package main

import (
    "fmt"
    "time"
)

func descargar(duracion time.Duration) <-chan string {
    ch := make(chan string)
    go func() {
        time.Sleep(duracion)
        ch <- "✓ Descarga completa"
    }()
    return ch
}

func main() {
    // TIMEOUT CORTO (2s) pero descarga tarda 5s
    fmt.Println("Intentando descarga 5s con timeout 2s...")
    select {
    case msg := <-descargar(5 * time.Second):
        fmt.Println(msg)
    case <-time.After(2 * time.Second):
        fmt.Println("❌ Timeout!")
    }
    
    // TIMEOUT LARGO (10s) y descarga tarda 2s
    fmt.Println("\nIntentando descarga 2s con timeout 10s...")
    select {
    case msg := <-descargar(2 * time.Second):
        fmt.Println(msg)
    case <-time.After(10 * time.Second):
        fmt.Println("❌ Timeout!")
    }
}
```

**Salida:**
```
Intentando descarga 5s con timeout 2s...
❌ Timeout!

Intentando descarga 2s con timeout 10s...
✓ Descarga completa
```

---

## Comparación con p2pollo

```go
// EN P2POLLO: play.go línea 210-230
timeout := time.After(10 * time.Minute)
ticker := time.NewTicker(500 * time.Millisecond)

for !bufferReady {
    select {
    case <-timeout:  // ← Si pasan 10 minutos
        close(stopCopy)
        return  // Salir con error
        
    case <-ticker.C:  // ← Cada 500ms
        stat, _ := os.Stat(tmpPath)
        progress := (stat.Size() / targetSize) * 100
        fmt.Printf("Buffer: %d%%", progress)
    }
}

// ESTE PATRÓN USA:
// - time.After() para timeout
// - time.NewTicker() para actualizaciones periódicas
// - select para esperar múltiples eventos
// - Exactamente como Ejercicio 10!
```

---

## Ejercicio Final: Mini Streaming (versión simplificada)

```go
package main

import (
    "fmt"
    "time"
)

// Simula descarga de archivo (anacrolix)
func simularDescarga(tamaño int) <-chan int {
    progress := make(chan int)
    go func() {
        for i := 0; i <= 100; i++ {
            progress <- i
            time.Sleep(50 * time.Millisecond)
        }
        close(progress)
    }()
    return progress
}

// Simula copia a archivo temporal
func copiar(progress <-chan int) <-chan string {
    estado := make(chan string)
    go func() {
        for p := range progress {
            estado <- fmt.Sprintf("Copiado: %d%%", p)
        }
        close(estado)
    }()
    return estado
}

// Monitorea descarga
func monitorear(progress <-chan int) <-chan string {
    status := make(chan string)
    go func() {
        for p := range progress {
            if p%25 == 0 {
                status <- fmt.Sprintf("📊 Descargado: %d%%", p)
            }
        }
        close(status)
    }()
    return status
}

func main() {
    // Lanzar descarga
    descarga := simularDescarga(500) // 500MB
    
    // Fan-out: uno para copia, uno para monitoreo
    copia := copiar(descarga)
    monitor := monitorear(descarga)
    
    // Fan-in: combinar ambos outputs
    done := make(chan bool)
    go func() {
        for {
            select {
            case msg := <-copia:
                if msg != "" {
                    fmt.Println(msg)
                } else {
                    fmt.Println("✓ Copia completada")
                }
                
            case msg := <-monitor:
                if msg != "" {
                    fmt.Println(msg)
                } else {
                    fmt.Println("✓ Monitoreo completado")
                    done <- true
                    return
                }
            }
        }
    }()
    
    <-done
}
```

---

## Conclusión

Estos ejercicios cubren los conceptos fundamentales:

| Concepto | Ejercicio | p2pollo |
|----------|-----------|---------|
| Básico | 1 | `stopCopy` canal |
| Buffer | 2 | Archivo temporal como buffer |
| Select | 3 | Loop de monitoreo con select |
| WaitGroup | 4 | (no usado directamente) |
| Worker Pool | 5 | Goroutine de copia |
| Context | 6 | (podría mejorar con context) |
| Mutex | 7 | `sync.RWMutex` en client.go |
| Directional Ch. | 8 | Tipo-safety en funciones |
| Fan-Out/In | 9 | Copia + Monitoreo en paralelo |
| Timeout | 10 | Buffer timeout y descarga timeout |

¡Practica estos ejercicios y entenderás Go concurrencia perfectamente! 🚀

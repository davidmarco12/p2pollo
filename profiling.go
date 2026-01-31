package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

// EnableProfiling inicia el servidor de profiling en puerto 6060
// Esto permite usar:
//
//	go tool pprof http://localhost:6060/debug/pprof/heap      (memoria)
//	go tool pprof http://localhost:6060/debug/pprof/profile   (CPU)
//	go tool pprof http://localhost:6060/debug/pprof/goroutine (goroutines)
//	go tool pprof http://localhost:6060/debug/pprof/mutex     (locks)
func EnableProfiling() {
	go func() {
		fmt.Println("🔍 Profiling habilitado en http://localhost:6060/debug/pprof/")
		fmt.Println("   Usa: go tool pprof http://localhost:6060/debug/pprof/heap")

		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Printf("Error iniciando pprof server: %v", err)
		}
	}()
}

// PrintMemoryStats imprime estadísticas de memoria actuales
func PrintMemoryStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	fmt.Println("\n📊 =====  ESTADÍSTICAS DE MEMORIA  =====")
	fmt.Printf("   Alloc:        %v MB\n", bToMb(m.Alloc))
	fmt.Printf("   TotalAlloc:   %v MB\n", bToMb(m.TotalAlloc))
	fmt.Printf("   Sys:          %v MB\n", bToMb(m.Sys))
	fmt.Printf("   NumGC:        %v\n", m.NumGC)
	fmt.Printf("   Goroutines:   %v\n", runtime.NumGoroutine())
	fmt.Println("========================================\n")
}

// PrintGoroutineStats imprime info de goroutines
func PrintGoroutineStats() {
	fmt.Println("\n🔄 =====  ESTADÍSTICAS DE GOROUTINES  =====")
	fmt.Printf("   Activas: %d\n", runtime.NumGoroutine())
	fmt.Println("==========================================\n")
}

// bToMb convierte bytes a megabytes
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

// MonitorMemory monitorea memoria cada N segundos
func MonitorMemory(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			PrintMemoryStats()
		}
	}()
}

// ProfileCPU inicia profiling de CPU por N segundos
// Uso: defer ProfileCPU(30 * time.Second)()
func ProfileCPU(duration time.Duration) func() {
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal("Could not create CPU profile: ", err)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("Could not start CPU profile: ", err)
	}

	fmt.Printf("📊 CPU Profiling iniciado por %v segundos\n", duration)
	fmt.Println("   Después ejecuta: go tool pprof cpu.prof")

	time.AfterFunc(duration, func() {
		pprof.StopCPUProfile()
		f.Close()
		fmt.Println("✓ CPU profile guardado en cpu.prof")
	})

	return func() {
		pprof.StopCPUProfile()
		f.Close()
	}
}

// ProfileMemory escribe un snapshot de memoria
// Uso: ProfileMemory("mem.prof")
func ProfileMemory(filename string) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatal("Could not create memory profile: ", err)
	}
	defer f.Close()

	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("Could not write memory profile: ", err)
	}

	fmt.Printf("✓ Memory profile guardado en %s\n", filename)
	fmt.Printf("   Después ejecuta: go tool pprof %s\n", filename)
}

// Ejemplo de uso en main
/*
func main() {
	// Opción 1: Profiling en vivo (HTTP)
	EnableProfiling()

	// Opción 2: Monitorear memoria cada 5 segundos
	MonitorMemory(5 * time.Second)

	// Opción 3: CPU profiling por 30 segundos
	// defer ProfileCPU(30 * time.Second)()

	// Tu código aquí
	time.Sleep(1 * time.Minute)

	// Opción 4: Snapshot de memoria
	ProfileMemory("mem.prof")
}
*/

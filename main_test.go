package main

import (
	"testing"
	"time"
)

// TestPlaceholder - Test básico para verificar que el programa compila
func TestPlaceholder(t *testing.T) {
	// Este es un placeholder para que el paquete main tenga tests
	if true != true {
		t.Fatal("Esto nunca debería fallar")
	}
}

// BenchmarkSleep - Benchmark simple para ver cómo funciona
func BenchmarkSleep(b *testing.B) {
	for i := 0; i < b.N; i++ {
		time.Sleep(1 * time.Millisecond)
	}
}

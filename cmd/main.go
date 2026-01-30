package main

import (
	"fmt"
	"os"
)

var (
	version = "dev"
)

func main() {
	fmt.Printf("Streaming CLI v%s\n", version)
	fmt.Println("Iniciando aplicación...")
	os.Exit(0)
}

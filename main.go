package main

import (
	_ "github.com/armanceau/go-url-shortener/cmd/cli"    // Importe le package 'cli' pour que ses init() soient exécutés
	_ "github.com/armanceau/go-url-shortener/cmd/server" // Importe le package 'server' pour que ses init() soient exécutés
)

func main() {
	// TODO
}

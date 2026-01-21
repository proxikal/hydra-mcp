package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/proxikal/hydra/internal/proxy"
	"github.com/proxikal/hydra/internal/watcher"
)

func main() {
	watchPattern := flag.String("watch", ".", "Directory to watch")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: hydra [options] -- <command> [args...]")
		os.Exit(1)
	}

	log.SetPrefix("[Hydra] ")
	log.Printf("Starting supervisor for: %v", args)

	// Initialize Proxy
	p := proxy.New(args)

	// Start initial process
	if err := p.Start(); err != nil {
		log.Fatalf("Failed to start initial process: %v", err)
	}

	// Start Watcher
	go func() {
		if err := watcher.Watch(*watchPattern, func() {
			log.Println("Changes detected. Triggering restart...")
			p.Restart()
		}); err != nil {
			log.Printf("Watcher error: %v", err)
		}
	}()

	// Pipe Stdin -> Proxy
	// This blocks until stdin closes (Agent disconnects)
	go func() {
		if _, err := io.Copy(p, os.Stdin); err != nil {
			log.Printf("Stdin pipe error: %v", err)
		}
	}()

	// Wait for interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down...")
}

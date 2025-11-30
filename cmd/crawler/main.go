package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Davis1233798/crawler-go/internal/browser"
	"github.com/Davis1233798/crawler-go/internal/config"
	"github.com/Davis1233798/crawler-go/internal/metrics"
	"github.com/Davis1233798/crawler-go/internal/proxy"
	"github.com/Davis1233798/crawler-go/internal/vless"
)

func main() {
	cfg := config.GetConfig()

	log.Println("Starting Crawler (Go Version) - VLESS SUPPORT MODE")
	log.Printf("Threads: %d, Duration: %ds, Headless: %v", cfg.Threads, cfg.Duration, cfg.Headless)

	// Start Metrics
	metrics.StartMetricsServer(cfg.MetricsPort)

	// VLESS Setup
	vlessContent, err := os.ReadFile("vless.txt")
	if err != nil {
		log.Fatalf("Failed to read vless.txt: %v. Please provide a valid VLESS URI in vless.txt", err)
	}
	vlessStr := strings.TrimSpace(string(vlessContent))

	vm := vless.NewManager("xray.exe", 10808) // Assuming xray.exe in cwd or path, port 10808
	vConfig, err := vm.ParseVless(vlessStr)
	if err != nil {
		log.Fatalf("Failed to parse VLESS URI: %v", err)
	}

	if err := vm.GenerateConfig(vConfig); err != nil {
		log.Fatalf("Failed to generate Xray config: %v", err)
	}

	if err := vm.Start(); err != nil {
		log.Fatalf("Failed to start Xray: %v. Ensure xray.exe is available.", err)
	}
	defer vm.Stop()

	log.Println("Xray started on 127.0.0.1:10808")

	// Create a single proxy object pointing to local Xray
	localProxy := &proxy.Proxy{
		Server: "socks5://127.0.0.1:10808",
	}

	// Init Browser Pool
	browserPool := browser.NewBrowserPool(cfg.Headless)
	if err := browserPool.Initialize(); err != nil {
		log.Fatalf("Failed to initialize browser pool: %v", err)
	}
	defer browserPool.Shutdown()

	// Worker Pool
	var wg sync.WaitGroup
	tasks := make(chan struct{}, cfg.Threads)

	stopChan := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		close(stopChan)
	}()

	log.Println("Workers started...")

	// Main loop
	log.Println("Starting batch processing...")

loop:
	for {
		select {
		case <-stopChan:
			break loop
		default:
			select {
			case tasks <- struct{}{}:
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() { <-tasks }()

					bot := browser.NewBrowserBot(browserPool)

					log.Println("Starting batch (VLESS Proxy)")

					metrics.ActiveThreads.Inc()
					start := time.Now()

					// Use local VLESS proxy
					err := bot.RunBatch(cfg.Targets, localProxy, cfg.Duration)

					duration := time.Since(start).Seconds()
					metrics.SessionDuration.Observe(duration)
					metrics.ActiveThreads.Dec()

					if err != nil {
						log.Printf("Batch finished with error: %v", err)
					} else {
						log.Println("Batch completed successfully")
						metrics.TasksCompleted.Inc()
					}
				}()
			case <-stopChan:
				break loop
			}
		}
	}

	wg.Wait()
	log.Println("Shutdown complete.")
}

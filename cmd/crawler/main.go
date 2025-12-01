package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Davis1233798/crawler-go/internal/browser"
	"github.com/Davis1233798/crawler-go/internal/config"
	"github.com/Davis1233798/crawler-go/internal/metrics"
	"github.com/Davis1233798/crawler-go/internal/proxy"
)

func main() {
	cfg := config.GetConfig()

	log.Println("Starting Crawler (Go Version)")
	log.Printf("Threads: %d, Duration: %ds, Headless: %v", cfg.Threads, cfg.Duration, cfg.Headless)

	// Start Metrics
	metrics.StartMetricsServer(cfg.MetricsPort)

	// Init Proxy Pool
	proxyPool := proxy.NewMemoryProxyPool("proxies.txt", cfg.Threads*2)
	// Assuming targetURL from first target for verification
	targetURL := ""
	if len(cfg.Targets) > 0 {
		targetURL = cfg.Targets[0]
	}
	// Initial load from disk (VLESS only)
	proxyPool.Initialize(true, targetURL)
	log.Printf("Available proxies: %d", proxyPool.Size())
	defer proxyPool.SaveToDisk() // Save cleaned list on exit

	// Init Browser Manager
	browserManager := browser.NewBrowserManager(cfg.Headless)
	if err := browserManager.Initialize(); err != nil {
		log.Fatalf("Failed to initialize browser manager: %v", err)
	}
	defer browserManager.Shutdown()

	// Worker Pool
	var wg sync.WaitGroup
	tasks := make(chan struct{}, cfg.Threads) // Semaphore channel

	stopChan := make(chan struct{})
	// Signal handling
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
			// Try to start a task if slots available
			select {
			case tasks <- struct{}{}:
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() { <-tasks }()

					// 1. Get Proxy (Exclusive)
					p := proxyPool.GetProxy()
					if p == nil {
						// No free proxies, wait and retry
						time.Sleep(2 * time.Second)
						return
					}

					// 2. Create Bot with Manager
					bot := browser.NewBrowserBot(browserManager)

					// 3. Run Batch (Launches fresh browser, runs, closes)
					log.Printf("Using proxy %s for batch", p.Server)
					err := bot.RunBatch(cfg.Targets, p, cfg.Duration)
					
					// 4. Release Proxy
					proxyPool.ReleaseProxy(p)

					if err != nil {
						log.Printf("Batch finished with error: %v", err)
						proxyPool.MarkFailed(*p)
					} else {
						log.Println("Batch completed successfully")
						metrics.TasksCompleted.Inc()
					}
					
					// 5. Wait before next iteration (optional)
					time.Sleep(1 * time.Second)
				}()
				
				// Small delay to stagger starts
				time.Sleep(500 * time.Millisecond)
				
			case <-stopChan:
				break loop
			}
		}
	}

	wg.Wait()
	log.Println("Shutdown complete.")
}

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
	defer proxyPool.SaveToDisk() // Save cleaned list on exit

	// Init Browser Pool
	browserPool := browser.NewBrowserPool(cfg.Headless)
	if err := browserPool.Initialize(); err != nil {
		log.Fatalf("Failed to initialize browser pool: %v", err)
	}
	defer browserPool.Shutdown()

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

					// Each worker gets its own bot instance, which will acquire a browser from the pool
					bot := browser.NewBrowserBot(browserPool)

					// Acquire a proxy (exclusive lock)
					p := proxyPool.GetProxy()
					if p == nil {
						// No free proxies, wait and retry
						time.Sleep(1 * time.Second)
						return
					}
					defer proxyPool.ReleaseProxy(p)

					log.Printf("Using proxy %s for batch", p.String())

					metrics.ActiveThreads.Inc()

					start := time.Now()
					// RunBatch opens all targets
					err := bot.RunBatch(cfg.Targets, p, cfg.Duration)
					duration := time.Since(start).Seconds()
					metrics.SessionDuration.Observe(duration)

					metrics.ActiveThreads.Dec()

					if err != nil {
						log.Printf("Batch finished with error: %v", err)
						proxyPool.MarkFailed(*p)
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

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

	// Fetch proxies if pool is empty or small
	fetcher := proxy.NewProxyFetcher()
	// Initial load from disk
	proxyPool.Initialize(true, targetURL)
	defer proxyPool.SaveToDisk() // Save cleaned list on exit

	if proxyPool.Size() < cfg.Threads {
		log.Println("Proxy pool is low, fetching from APIs...")
		newProxies := fetcher.FetchAll(100)
		proxyPool.AddProxies(newProxies)
		// Save to disk after adding
		proxyPool.SaveToDisk()
		log.Printf("Proxy pool now has %d proxies", proxyPool.Size())
	}

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

	// Fetch Lock to prevent thundering herd
	var fetchLock sync.Mutex
	var isFetching bool

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

					// Acquire a proxy
					p := proxyPool.GetProxy()
					if p == nil {
						// Check if fetching
						fetchLock.Lock()
						if isFetching {
							fetchLock.Unlock()
							log.Println("Another worker is fetching proxies, waiting...")
							time.Sleep(2 * time.Second)
							return
						}
						
						// We are the fetcher
						isFetching = true
						fetchLock.Unlock()
						
						log.Println("No available proxies (all used or exhausted). Fetching free proxies...")
						
						defer func() {
							fetchLock.Lock()
							isFetching = false
							fetchLock.Unlock()
						}()
						
						// Fetch new proxies
						newProxies := fetcher.FetchAll(50)
						log.Printf("Fetched %d potential proxies. Verifying...", len(newProxies))
						
						// Verify fast
						var validProxies []string
						for _, np := range newProxies {
							parsed := proxy.ParseProxy(np)
							if parsed != nil && proxyPool.CheckProxyFast(*parsed) {
								validProxies = append(validProxies, np)
							}
						}
						
						if len(validProxies) > 0 {
							log.Printf("Found %d working free proxies. Adding to pool...", len(validProxies))
							proxyPool.AddProxies(validProxies) // This will add them to working list (and implicitly allow reuse since they are new hosts)
							proxyPool.SaveToDisk()
							// Try get again
							p = proxyPool.GetProxy()
						}
						
						if p == nil {
							log.Println("Still no proxies available. Sleeping 10s...")
							time.Sleep(10 * time.Second)
							return
						}
					}

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

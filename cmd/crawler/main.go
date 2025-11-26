package main

import (
	"log"
	"math/rand"
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
	
	// Seed random number generator once
	rand.Seed(time.Now().UnixNano())

	// Assuming targetURL from first target for verification
	targetURL := ""
	if len(cfg.Targets) > 0 {
		targetURL = cfg.Targets[0]
	}

	// Fetch proxies if pool is empty or small
	fetcher := proxy.NewProxyFetcher()
	// Initial load from disk
	proxyPool.Initialize(true, targetURL)

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

	bot := browser.NewBrowserBot(browserPool)

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

	// Task counter for injecting no-proxy requests
	taskCounter := 0
	var counterLock sync.Mutex

	// Main loop
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

					// Increment counter and check if we should run without proxy
					counterLock.Lock()
					taskCounter++
					useProxy := taskCounter%cfg.Threads != 0 // Every Nth task runs without proxy
					counterLock.Unlock()

					var p *proxy.Proxy
					if useProxy {
						// Get Proxy
						p = proxyPool.GetProxy(cfg.MaxViewsPerIP)
						if p == nil {
							log.Println("No proxies available (limit reached or empty), waiting...")
							time.Sleep(5 * time.Second)
							return
						}
						proxyPool.RecordUsage(p)
					} else {
						log.Println("Running task without proxy (direct connection)")
					}

					// Get Targets
					targets := make([]string, len(cfg.Targets))
					copy(targets, cfg.Targets)
					
					if len(targets) == 0 {
						log.Println("No targets available!")
						time.Sleep(5 * time.Second)
						return
					}

					// Shuffle targets to avoid all workers hitting the same URL at once
					rand.Shuffle(len(targets), func(i, j int) {
						targets[i], targets[j] = targets[j], targets[i]
					})

					metrics.ActiveThreads.Inc()
					defer metrics.ActiveThreads.Dec()

					start := time.Now()
					err := bot.Run(targets, p, cfg.Duration)
					duration := time.Since(start).Seconds()
					metrics.SessionDuration.Observe(duration)

					if err != nil {
						log.Printf("Task failed: %v", err)
						metrics.TasksFailed.Inc()
						if p != nil {
							proxyPool.MarkFailed(*p)
						}
						// Add a small delay on failure to prevent rapid looping if browser is crashing
						time.Sleep(2 * time.Second)
					} else {
						log.Printf("Task completed for %d targets", len(targets))
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

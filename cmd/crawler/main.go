package main

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Davis1233798/crawler-go/internal/browser"
	"github.com/Davis1233798/crawler-go/internal/config"
	"github.com/Davis1233798/crawler-go/internal/metrics"
	"github.com/Davis1233798/crawler-go/internal/notify"
	"github.com/Davis1233798/crawler-go/internal/proxy"
)

func main() {
	cfg := config.GetConfig()

	log.Println("Starting Crawler (Go Version)")
	log.Printf("Threads: %d, Duration: %ds, Headless: %v", cfg.Threads, cfg.Duration, cfg.Headless)

	// Init Notify
	notify.Init(cfg.DiscordWebhookURL)
	notify.Send("🚀 Crawler Started")

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

	// Periodic IP Fetcher
	if cfg.BaseVLESSLink != "" && len(cfg.PreferredIPAPIs) > 0 {
		go func() {
			ticker := time.NewTicker(6 * time.Hour)
			defer ticker.Stop()

			updateIPs := func() {
				log.Println("🔄 Fetching preferred IPs...")
				ips, err := proxy.FetchPreferredIPs(cfg.PreferredIPAPIs)
				if err != nil {
					log.Printf("❌ Failed to fetch IPs: %v", err)
					return
				}
				if len(ips) > 0 {
					proxyPool.UpdateProxiesFromIPs(cfg.BaseVLESSLink, ips)
				}
			}

			// Run immediately
			updateIPs()

			for {
				select {
				case <-ticker.C:
					updateIPs()
				case <-stopChan:
					return
				}
			}
		}()
	}

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
						
						// If timeout, we ran for full duration, so restart immediately (no penalty)
						if strings.Contains(err.Error(), "timed out") {
							log.Println("🔄 Timeout detected, restarting immediately...")
							time.Sleep(100 * time.Millisecond)
						} else {
							// Penalty delay on fast failure to prevent storm
							delay := time.Duration(rand.Intn(10000)+10000) * time.Millisecond
							log.Printf("⚠️ Thread sleeping for %v after failure...", delay)
							time.Sleep(delay)
						}
					} else {
						log.Println("Batch completed successfully")
						metrics.TasksCompleted.Inc()
						// Normal delay
						time.Sleep(time.Duration(rand.Intn(4000)+1000) * time.Millisecond)
					}
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

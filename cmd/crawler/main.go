package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
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

	if !cfg.NoProxyMode {
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
	} else {
		log.Println("Running in NO_PROXY_MODE. Skipping proxy initialization.")
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
	tasksScheduled := 0
loop:
	for {
		if cfg.RunOnce && tasksScheduled >= cfg.Threads {
			log.Println("RunOnce mode: All tasks scheduled. Waiting for completion...")
			break loop
		}

		select {
		case <-stopChan:
			break loop
		default:
			// Try to start a task if slots available
			select {
			case tasks <- struct{}{}:
				tasksScheduled++
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() { <-tasks }()

					// Increment counter and check if we should run without proxy
					counterLock.Lock()
					taskCounter++
					useProxy := taskCounter%cfg.Threads != 0 // Every Nth task runs without proxy
					if cfg.NoProxyMode {
						useProxy = false
					}
					counterLock.Unlock()

					var p *proxy.Proxy
					if useProxy {
						// Get Proxy
						p = proxyPool.GetProxy()
						if p == nil {
							log.Println("No proxies available, waiting...")
							time.Sleep(5 * time.Second)
							return
						}
					} else {
						log.Println("Running task without proxy (direct connection)")
					}

					// Get Target
					url := cfg.GetRandomTarget()
					if url == "" {
						log.Println("No targets available!")
						time.Sleep(5 * time.Second)
						return
					}

					metrics.ActiveThreads.Inc()
					defer metrics.ActiveThreads.Dec()

					var err error
					maxRetries := 3

					for i := 0; i < maxRetries; i++ {
						// Acquire proxy if needed
						if useProxy && p == nil {
							p = proxyPool.GetProxy()
							if p == nil {
								log.Println("No proxies available, waiting...")
								time.Sleep(5 * time.Second)
								// If we can't get a proxy, we can't proceed with this attempt
								continue
							}
						}

						start := time.Now()
						err = bot.Run(url, p, cfg.Duration)
						duration := time.Since(start).Seconds()
						metrics.SessionDuration.Observe(duration)

						if err == nil {
							log.Printf("Task completed for %s", url)
							metrics.TasksCompleted.Inc()
							break // Success, exit retry loop
						}

						// Handle failure
						log.Printf("Attempt %d/%d failed for %s: %v", i+1, maxRetries, url, err)

						if p != nil {
							proxyPool.MarkFailed(*p)
							p = nil // Reset proxy so we get a new one next time
						}

						if !useProxy {
							// If not using proxy, retrying might not help if site is down, but let's try once more or break
							break
						}

						// Small delay before retry
						time.Sleep(2 * time.Second)
					}

					if err != nil {
						log.Printf("Task permanently failed after %d attempts: %v", maxRetries, err)
						metrics.TasksFailed.Inc()
					}
				}()
			case <-stopChan:
				break loop
			}
		}
	}

	wg.Wait()
	log.Println("All tasks completed.")

	if cfg.SelfDestruct {
		selfDestruct()
	}

	log.Println("Shutdown complete.")
}

func selfDestruct() {
	log.Println("Self-destruct sequence initiated...")
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("Error getting hostname: %v", err)
		return
	}

	// We assume the hostname is the instance name.
	// We try to delete the instance. We need the zone.
	// We can try to fetch zone from metadata server, or just try without it (gcloud might prompt or fail).
	// To make it robust, let's try to get zone from metadata.

	zone := fetchZone()
	args := []string{"compute", "instances", "delete", hostname, "--quiet"}
	if zone != "" {
		args = append(args, "--zone", zone)
	}

	log.Printf("Executing: gcloud %s", strings.Join(args, " "))
	cmd := exec.Command("gcloud", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error self-destructing: %v\nOutput: %s", err, string(output))
	} else {
		log.Println("Self-destruct command issued successfully.")
	}
}

func fetchZone() string {
	// curl -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/zone
	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/instance/zone", nil)
	req.Header.Add("Metadata-Flavor", "Google")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// Zone is returned as "projects/PROJECT_NUM/zones/ZONE"
	parts := strings.Split(string(body), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

package proxy

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Proxy struct {
	Server   string
	Username string
	Password string
}

func (p Proxy) String() string {
	if p.Username != "" {
		// Reconstruct string for saving/logging
		// Assuming http/https, we strip protocol for the simple format usually
		// But let's keep it robust.
		// If original was ip:port:user:pass
		u, _ := url.Parse(p.Server)
		host := u.Host
		return fmt.Sprintf("%s:%s:%s", host, p.Username, p.Password)
	}
	return p.Server
}

func (p Proxy) ToURL() string {
	if p.Username != "" {
		u, _ := url.Parse(p.Server)
		return fmt.Sprintf("%s://%s:%s@%s", u.Scheme, p.Username, p.Password, u.Host)
	}
	return p.Server
}

type MemoryProxyPool struct {
	cacheFile      string
	minPoolSize    int
	workingProxies []Proxy
	failedProxies  map[string]bool
	lock           sync.RWMutex
}

func NewMemoryProxyPool(cacheFile string, minPoolSize int) *MemoryProxyPool {
	return &MemoryProxyPool{
		cacheFile:     cacheFile,
		minPoolSize:   minPoolSize,
		failedProxies: make(map[string]bool),
	}
}

func (p *MemoryProxyPool) Initialize(strictVerify bool, targetURL string) {
	log.Println("Initializing Memory Proxy Pool...")
	rawProxies := p.loadFromDisk()

	if len(rawProxies) > 0 && strictVerify {
		log.Println("Strictly verifying proxies...")
		p.workingProxies = p.verifyBatch(rawProxies, targetURL)
		log.Printf("✅ %d/%d proxies passed strict verification", len(p.workingProxies), len(rawProxies))
	} else {
		p.workingProxies = rawProxies
	}

	// Initial save to clean up bad proxies from file if verified
	if strictVerify {
		p.SaveToDisk()
	}
}

func (p *MemoryProxyPool) loadFromDisk() []Proxy {
	file, err := os.Open(p.cacheFile)
	if err != nil {
		log.Printf("Failed to load cache file: %v", err)
		return []Proxy{}
	}
	defer file.Close()

	var proxies []Proxy
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			if proxy := ParseProxy(line); proxy != nil {
				proxies = append(proxies, *proxy)
			}
		}
	}
	return proxies
}

func ParseProxy(proxyStr string) *Proxy {
	proxyStr = strings.TrimSpace(proxyStr)
	if strings.Contains(proxyStr, "://") {
		return &Proxy{Server: proxyStr}
	}

	parts := strings.Split(proxyStr, ":")
	if len(parts) == 4 {
		return &Proxy{
			Server:   fmt.Sprintf("http://%s:%s", parts[0], parts[1]),
			Username: parts[2],
			Password: parts[3],
		}
	} else if len(parts) == 2 {
		return &Proxy{
			Server: fmt.Sprintf("http://%s:%s", parts[0], parts[1]),
		}
	}
	return nil
}

func (p *MemoryProxyPool) GetProxy() *Proxy {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if len(p.workingProxies) == 0 {
		return nil
	}
	return &p.workingProxies[rand.Intn(len(p.workingProxies))]
}

func (p *MemoryProxyPool) MarkFailed(proxy Proxy) {
	p.lock.Lock()
	defer p.lock.Unlock()

	proxyStr := proxy.String()
	// Remove from working
	for i, px := range p.workingProxies {
		if px.String() == proxyStr { // Simple comparison
			p.workingProxies = append(p.workingProxies[:i], p.workingProxies[i+1:]...)
			break
		}
	}
	p.failedProxies[proxyStr] = true
}

func (p *MemoryProxyPool) verifyBatch(proxies []Proxy, targetURL string) []Proxy {
	var verified []Proxy
	var wg sync.WaitGroup
	results := make(chan Proxy, len(proxies))
	sem := make(chan struct{}, 50) // Semaphore for concurrency limit

	for _, px := range proxies {
		wg.Add(1)
		go func(proxy Proxy) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if checkProxy(proxy, targetURL) {
				results <- proxy
			}
		}(px)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for px := range results {
		verified = append(verified, px)
	}
	return verified
}

func checkProxy(proxy Proxy, targetURL string) bool {
	if targetURL == "" {
		targetURL = "https://httpbin.org/ip"
	}

	proxyURL, err := url.Parse(proxy.ToURL())
	if err != nil {
		return false
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(targetURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

func (p *MemoryProxyPool) AddProxies(proxies []string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	addedCount := 0
	for _, proxyStr := range proxies {
		// Check if already exists (simple check)
		exists := false
		for _, existing := range p.workingProxies {
			if existing.String() == proxyStr {
				exists = true
				break
			}
		}
		if exists {
			continue
		}

		if proxy := ParseProxy(proxyStr); proxy != nil {
			p.workingProxies = append(p.workingProxies, *proxy)
			addedCount++
		}
	}
	if addedCount > 0 {
		log.Printf("Added %d new proxies to pool. Total: %d", addedCount, len(p.workingProxies))
	}
}

func (p *MemoryProxyPool) SaveToDisk() {
	p.lock.RLock()
	proxies := make([]Proxy, len(p.workingProxies))
	copy(proxies, p.workingProxies)
	p.lock.RUnlock()

	file, err := os.Create(p.cacheFile)
	if err != nil {
		log.Printf("Error saving proxies: %v", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, px := range proxies {
		writer.WriteString(px.String() + "\n")
	}
	writer.Flush()
}

func (p *MemoryProxyPool) Size() int {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return len(p.workingProxies)
}

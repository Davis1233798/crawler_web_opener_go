package proxy

import (
	"bufio"
	"encoding/json"
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
	usageMap       map[string]time.Time // Last used time for each IP (by host)
	usageFile      string
}

func NewMemoryProxyPool(cacheFile string, minPoolSize int) *MemoryProxyPool {
	p := &MemoryProxyPool{
		cacheFile:     cacheFile,
		minPoolSize:   minPoolSize,
		workingProxies: []Proxy{},
		failedProxies: make(map[string]bool),
		usageMap:      make(map[string]time.Time),
		usageFile:     "ip_usage.json",
	}
	p.loadUsage()
	return p
}

func (p *MemoryProxyPool) loadUsage() {
	if _, err := os.Stat(p.usageFile); os.IsNotExist(err) {
		return
	}
	file, err := os.Open(p.usageFile)
	if err != nil {
		log.Printf("Failed to open usage file: %v", err)
		return
	}
	defer file.Close()
	
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&p.usageMap); err != nil {
		log.Printf("Failed to decode usage file: %v", err)
	}
}

func (p *MemoryProxyPool) saveUsage() {
	file, err := os.Create(p.usageFile)
	if err != nil {
		log.Printf("Failed to create usage file: %v", err)
		return
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(p.usageMap); err != nil {
		log.Printf("Failed to encode usage file: %v", err)
	}
}

func (p *MemoryProxyPool) Initialize(strictVerify bool, targetURL string) {
	log.Println("Initializing Memory Proxy Pool...")
	rawProxies := p.loadFromDisk()

	if len(rawProxies) > 0 && strictVerify {
		log.Println("Strictly verifying proxies...")
		p.workingProxies = p.VerifyBatch(rawProxies, targetURL)
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
	seen := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			if proxy := ParseProxy(line); proxy != nil {
				proxyStr := proxy.String()
				if !seen[proxyStr] {
					proxies = append(proxies, *proxy)
					seen[proxyStr] = true
				}
			}
		}
	}
	log.Printf("Loaded %d unique proxies from disk (deduplicated)", len(proxies))
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
	p.lock.Lock()
	defer p.lock.Unlock()

	// 1. Try to find a proxy from workingProxies that hasn't been used today
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	
	var available []*Proxy
	var allWorking []*Proxy

	for i := range p.workingProxies {
		proxy := &p.workingProxies[i]
		allWorking = append(allWorking, proxy)

		host := getHost(proxy.Server)
		lastUsed, ok := p.usageMap[host]
		if !ok || lastUsed.Before(today) {
			available = append(available, proxy)
		}
	}
	
	// BEST EFFORT: If we have fresh proxies, use them.
	if len(available) > 0 {
		selected := available[rand.Intn(len(available))]
		host := getHost(selected.Server)
		p.usageMap[host] = now
		p.saveUsage()
		return selected
	}

	// FALLBACK: If we have NO fresh proxies but DO have working proxies, 
	// reuse one to keep the crawler alive (Log a warning)
	if len(allWorking) > 0 {
		log.Println("⚠️ No fresh IPs available for today. Reusing existing proxy to maintain activity.")
		selected := allWorking[rand.Intn(len(allWorking))]
		host := getHost(selected.Server)
		p.usageMap[host] = now
		p.saveUsage()
		return selected
	}
	
	return nil
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

func (p *MemoryProxyPool) VerifyBatch(proxies []Proxy, targetURL string) []Proxy {
	var verified []Proxy
	var wg sync.WaitGroup
	results := make(chan Proxy, len(proxies))
	sem := make(chan struct{}, 200) // Increased semaphore for faster batch processing

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
		// httpbin.org often rate limits. Use Google for robust connectivity check.
		targetURL = "http://www.google.com"
	}

	proxyURL, err := url.Parse(proxy.ToURL())
	if err != nil {
		return false
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			DisableKeepAlives: true,
		},
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return false
	}
	// Add User-Agent to avoid simple bot filtering
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		if resp.StatusCode == 429 {
			log.Printf("Verification 429 (Rate Limit) for %s on %s", proxy.Server, targetURL)
		}
		return false
	}

	return true
}

func (p *MemoryProxyPool) CheckProxyFast(proxy Proxy) bool {
	// Reusing the global checkProxy function
	return checkProxy(proxy, "https://httpbin.org/ip")
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

func getHost(server string) string {
	u, err := url.Parse(server)
	if err != nil {
		return server
	}
	return u.Hostname()
}

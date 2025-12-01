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
	vlessAdapters  map[string]*VLESSAdapter
	lock           sync.RWMutex
}

func NewMemoryProxyPool(cacheFile string, minPoolSize int) *MemoryProxyPool {
	return &MemoryProxyPool{
		cacheFile:     cacheFile,
		minPoolSize:   minPoolSize,
		failedProxies: make(map[string]bool),
		vlessAdapters: make(map[string]*VLESSAdapter),
	}
}

func (p *MemoryProxyPool) Initialize(strictVerify bool, targetURL string) {
	log.Println("Initializing Memory Proxy Pool...")
	rawProxies := p.loadFromDisk()
	
	// Load VLESS proxies
	vlessProxies := p.loadVLESSFromDisk()
	for _, vp := range vlessProxies {
		if strings.HasPrefix(vp.Server, "vless://") {
			if _, ok := p.vlessAdapters[vp.Server]; !ok {
				adapter, err := StartVLESSAdapter(vp.Server)
				if err != nil {
					log.Printf("Failed to start adapter for %s: %v", vp.Server, err)
					continue
				}
				p.vlessAdapters[vp.Server] = adapter
				log.Printf("Started VLESS adapter at %s", adapter.SocksAddr())
			}
		}
	}
	rawProxies = append(rawProxies, vlessProxies...)

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

func (p *MemoryProxyPool) loadVLESSFromDisk() []Proxy {
	file, err := os.Open("vless.txt")
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to load vless.txt: %v", err)
		}
		return []Proxy{}
	}
	defer file.Close()

	var proxies []Proxy
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && strings.HasPrefix(line, "vless://") {
			// For VLESS, we treat the link itself as the "Server" initially
			// The adapter will be started when added to the pool
			proxies = append(proxies, Proxy{Server: line})
		}
	}
	return proxies
}

func ParseProxy(proxyStr string) *Proxy {
	proxyStr = strings.TrimSpace(proxyStr)
	if strings.HasPrefix(proxyStr, "vless://") {
		return &Proxy{Server: proxyStr}
	}
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
	
	// Select a random proxy
	proxy := p.workingProxies[rand.Intn(len(p.workingProxies))]
	
	// If it's a VLESS proxy, check/start adapter and return local SOCKS5 address
	if strings.HasPrefix(proxy.Server, "vless://") {
		// We need to upgrade the lock to write lock if we need to start an adapter
		// But we currently hold RLock. This is tricky.
		// For simplicity, let's assume adapters are started in AddProxies or Initialize.
		// But wait, AddProxies is where we should start them.
		
		if adapter, ok := p.vlessAdapters[proxy.Server]; ok {
			return &Proxy{Server: "socks5://" + adapter.SocksAddr()}
		} else {
			// Adapter not found? This shouldn't happen if logic is correct.
			// Maybe it failed to start?
			log.Printf("VLESS adapter not found for %s", proxy.Server)
			return nil // Or try to start it?
		}
	}
	
	return &proxy
}

func (p *MemoryProxyPool) MarkFailed(proxy Proxy) {
	p.lock.Lock()
	defer p.lock.Unlock()

	// If it was a local SOCKS5 proxy (from VLESS), we need to find the original VLESS link
	// But the `proxy` passed here might be the "socks5://127.0.0.1:..." one.
	// This makes mapping back difficult.
	// However, `bot.RunBatch` receives the proxy returned by `GetProxy`.
	// If `GetProxy` returns a transformed proxy, we lose the original ID.
	// We should probably change `GetProxy` to return the original proxy struct, 
	// and let the caller resolve the actual address? 
	// OR, we keep the original VLESS link in the Proxy struct but add a "ConnectAddress" field?
	// Given the existing struct is simple, let's try to match by value.
	
	// Actually, for VLESS, if it fails, maybe we shouldn't remove it immediately or we should restart the adapter?
	// For now, let's just remove it from working list.
	
	// Wait, if `proxy` is `socks5://127.0.0.1:xxxxx`, we can't easily find it in `workingProxies` 
	// because `workingProxies` stores `vless://...`.
	// We need a way to map back.
	
	// Let's modify Proxy struct to hold metadata? No, that changes API.
	// Let's iterate adapters to find which one matches the port?
	
	targetStr := proxy.String()
	var originalProxyStr string
	
	if strings.HasPrefix(targetStr, "socks5://127.0.0.1:") {
		// It's likely a VLESS adapter
		for vlessLink, adapter := range p.vlessAdapters {
			if "socks5://"+adapter.SocksAddr() == targetStr {
				originalProxyStr = vlessLink
				break
			}
		}
	} else {
		originalProxyStr = targetStr
	}
	
	if originalProxyStr == "" {
		return // Can't find it
	}

	proxyStr := originalProxyStr
	// Remove from working
	for i, px := range p.workingProxies {
		if px.String() == proxyStr { // Simple comparison
			p.workingProxies = append(p.workingProxies[:i], p.workingProxies[i+1:]...)
			break
		}
	}
	p.failedProxies[proxyStr] = true
	
	// If it was VLESS, stop the adapter
	if adapter, ok := p.vlessAdapters[proxyStr]; ok {
		adapter.Close()
		delete(p.vlessAdapters, proxyStr)
	}
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

			// For verification, if it's VLESS, we need to start a temp adapter or use existing?
			// Since this is Initialize/Add, we should start adapters here if not exists.
			// But `verifyBatch` is pure.
			// Let's modify `checkProxy` to handle VLESS startup/teardown if needed?
			// Or better: Start adapters BEFORE verification for all VLESS proxies.
			
			// This is getting complex. Let's simplify:
			// 1. `AddProxies` starts adapters.
			// 2. `verifyBatch` uses the started adapters.
			
			// But `verifyBatch` takes `[]Proxy`.
			// We need to ensure adapters are running.
			
			// Let's do a quick hack: In `checkProxy`, if VLESS, start temp adapter.
			if strings.HasPrefix(proxy.Server, "vless://") {
				// Start temp adapter
				adapter, err := StartVLESSAdapter(proxy.Server)
				if err != nil {
					log.Printf("Failed to start VLESS adapter for verification: %v", err)
					return
				}
				defer adapter.Close()
				
				// Create a temp proxy pointing to local socks
				tempProxy := Proxy{Server: "socks5://" + adapter.SocksAddr()}
				if checkProxy(tempProxy, targetURL) {
					results <- proxy
				}
			} else {
				if checkProxy(proxy, targetURL) {
					results <- proxy
				}
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
			// If VLESS, start adapter
			if strings.HasPrefix(proxy.Server, "vless://") {
				if _, ok := p.vlessAdapters[proxy.Server]; !ok {
					adapter, err := StartVLESSAdapter(proxy.Server)
					if err != nil {
						log.Printf("Failed to start adapter for %s: %v", proxy.Server, err)
						continue
					}
					p.vlessAdapters[proxy.Server] = adapter
					log.Printf("Started VLESS adapter at %s", adapter.SocksAddr())
				}
			}
			
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

	// Separate VLESS and normal proxies
	// We only save normal proxies to proxies.txt? 
	// Or we save all?
	// The user asked for vless.txt.
	// Let's save VLESS to vless.txt and others to proxies.txt
	
	file, err := os.Create(p.cacheFile)
	if err != nil {
		log.Printf("Error saving proxies: %v", err)
		return
	}
	defer file.Close()
	
	vlessFile, err := os.Create("vless.txt")
	if err != nil {
		log.Printf("Error saving vless proxies: %v", err)
	}
	defer vlessFile.Close()

	writer := bufio.NewWriter(file)
	vlessWriter := bufio.NewWriter(vlessFile)
	
	for _, px := range proxies {
		if strings.HasPrefix(px.Server, "vless://") {
			vlessWriter.WriteString(px.Server + "\n")
		} else {
			writer.WriteString(px.String() + "\n")
		}
	}
	writer.Flush()
	vlessWriter.Flush()
}

func (p *MemoryProxyPool) Size() int {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return len(p.workingProxies)
}

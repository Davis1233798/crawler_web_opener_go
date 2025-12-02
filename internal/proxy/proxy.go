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

	"github.com/Davis1233798/crawler-go/internal/notify"
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
	reserveProxies []Proxy // Proxies waiting to be used
	failedProxies  map[string]bool
	busyProxies    map[string]bool
	activeAdapters map[string]*VLESSAdapter // Map socksAddr -> Adapter
	proxyToVless   map[string]string      // Map socksAddr -> VLESS config string
	failureCounts  map[string]int
	lock           sync.RWMutex
}

func NewMemoryProxyPool(cacheFile string, minPoolSize int) *MemoryProxyPool {
	return &MemoryProxyPool{
		cacheFile:     cacheFile,
		minPoolSize:   minPoolSize,
		failedProxies: make(map[string]bool),
		activeAdapters: make(map[string]*VLESSAdapter),
		proxyToVless:   make(map[string]string),
		busyProxies:    make(map[string]bool),
		failureCounts:  make(map[string]int),
	}
}

// ... (Initialize remains same, skipping for brevity in replacement if possible, but I need to be careful with line numbers)
// Actually, I can just replace the struct and New function, and then add ReleaseProxy at the end or replace GetProxy.

// Let's replace struct and New first.
// Wait, I can't easily skip lines in ReplacementContent.
// I will replace the struct definition and NewMemoryProxyPool.

// Then I will replace GetProxy and add ReleaseProxy.

// Step 1: Replace struct and New
// Step 2: Replace GetProxy and add ReleaseProxy

// This tool call is for Step 1 & 2 combined if I target the right range.
// But GetProxy is further down.
// I'll do it in two chunks? No, replace_file_content is single chunk.
// I'll use multi_replace_file_content? No, "Do NOT make multiple parallel calls".
// I'll use replace_file_content for the struct/New, then another for GetProxy.

// Wait, I can use multi_replace_file_content.
// "Use this tool ONLY when you are making MULTIPLE, NON-CONTIGUOUS edits".
// Yes.

// Chunk 1: Struct and New
// Chunk 2: GetProxy and ReleaseProxy (ReleaseProxy is new, so I can append it or replace GetProxy and add it)


func (p *MemoryProxyPool) Initialize(strictVerify bool, targetURL string) {
	log.Println("Initializing Memory Proxy Pool (VLESS Only)...")
	
	// Load VLESS proxies only
	rawProxies := p.loadVLESSFromDisk()
	
	// Multiplexing: If we have fewer VLESS configs than minPoolSize,
	// we duplicate them to create multiple local adapters from the same config.
	// This allows concurrent connections via the same VLESS link.
	if len(rawProxies) > 0 && len(rawProxies) < p.minPoolSize {
		log.Printf("Multiplexing VLESS proxies: Have %d, Need %d", len(rawProxies), p.minPoolSize)
		originalCount := len(rawProxies)
		for len(rawProxies) < p.minPoolSize {
			// Round-robin selection from original proxies
			source := rawProxies[len(rawProxies)%originalCount]
			rawProxies = append(rawProxies, source)
		}
	}

	// We do NOT start adapters here anymore.
	// We just load the VLESS configs into workingProxies.
	// GetProxy will start them on demand.

	for _, vp := range rawProxies {
		if strings.HasPrefix(vp.Server, "vless://") {
			p.workingProxies = append(p.workingProxies, vp)
		}
	}
	
	log.Printf("Initialized pool with %d VLESS configs (Ephemeral mode)", len(p.workingProxies))
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
	p.lock.Lock()
	defer p.lock.Unlock()

	if len(p.workingProxies) == 0 {
		return nil
	}

	// Iterate through proxies to find a free one
	for _, proxy := range p.workingProxies {
		if !p.busyProxies[proxy.Server] {
			// Found a free VLESS config
			p.busyProxies[proxy.Server] = true

			// Start Ephemeral Adapter
			if strings.HasPrefix(proxy.Server, "vless://") {
				adapter, err := StartVLESSAdapter(proxy.Server)
				if err != nil {
					log.Printf("Failed to start ephemeral adapter for %s: %v", proxy.Server, err)
					// Mark as failed immediately? Or just skip?
					// Let's skip and try next
					delete(p.busyProxies, proxy.Server)
					continue
				}
				
				socksAddr := "socks5://" + adapter.SocksAddr()
				p.activeAdapters[socksAddr] = adapter
				p.proxyToVless[socksAddr] = proxy.Server
				
				return &Proxy{Server: socksAddr}
			}
			
			// Normal proxy (not VLESS) - not fully supported in this refactor but keeping logic
			return &proxy
		}
	}

	// All proxies are busy
	return nil
}

func (p *MemoryProxyPool) ReleaseProxy(proxy *Proxy) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if proxy == nil {
		return
	}

	socksAddr := proxy.Server
	
	// Check if it's an ephemeral adapter
	if adapter, ok := p.activeAdapters[socksAddr]; ok {
		// Close the adapter (Recycle)
		adapter.Close()
		delete(p.activeAdapters, socksAddr)
		
		// Unmark the VLESS config as busy
		if vlessLink, ok := p.proxyToVless[socksAddr]; ok {
			delete(p.busyProxies, vlessLink)
			delete(p.proxyToVless, socksAddr)
		}
	} else {
		// Normal proxy
		if p.busyProxies[socksAddr] {
			delete(p.busyProxies, socksAddr)
		}
	}
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
	
	socksAddr := proxy.String()
	
	// Handle Ephemeral Adapter
	if adapter, ok := p.activeAdapters[socksAddr]; ok {
		// Close it
		adapter.Close()
		delete(p.activeAdapters, socksAddr)
		
		// Handle VLESS failure logic
		if vlessLink, ok := p.proxyToVless[socksAddr]; ok {
			delete(p.proxyToVless, socksAddr)
			delete(p.busyProxies, vlessLink) // Release busy so we can remove/retry
			
			p.failureCounts[vlessLink]++
			if p.failureCounts[vlessLink] > 3 {
				log.Printf("❌ VLESS node failed too many times (%d). Removing: %s", p.failureCounts[vlessLink], vlessLink[:30]+"...")
				// Remove from workingProxies
				for i, px := range p.workingProxies {
					if px.Server == vlessLink {
						p.workingProxies = append(p.workingProxies[:i], p.workingProxies[i+1:]...)
						break
					}
				}
				p.failedProxies[vlessLink] = true
			} else {
				log.Printf("⚠️ VLESS node failed (Count: %d/3). Retrying next time.", p.failureCounts[vlessLink])
			}
		}
		return
	}

	// Normal Proxy Logic
	proxyStr := socksAddr
	for i, px := range p.workingProxies {
		if px.String() == proxyStr { 
			p.workingProxies = append(p.workingProxies[:i], p.workingProxies[i+1:]...)
			break
		}
	}
	p.failedProxies[proxyStr] = true
	
	p.replenish()
}

func (p *MemoryProxyPool) replenish() {
	p.lock.Lock()
	defer p.lock.Unlock()

	if len(p.workingProxies) >= p.minPoolSize {
		return
	}
	
	if len(p.reserveProxies) == 0 {
		log.Println("⚠️ No reserve proxies available to replenish pool.")
		return
	}

	// Rate limit replenishment to prevent storm
	time.Sleep(500 * time.Millisecond)
	
	// Pop from reserve
	newProxy := p.reserveProxies[0]
	p.reserveProxies = p.reserveProxies[1:]
	
	serverLog := newProxy.Server
	if len(serverLog) > 30 {
		serverLog = serverLog[:30] + "..."
	}
	log.Printf("Replenishing pool with reserve proxy: %s", serverLog)
	
	// Ephemeral: Just add to working, don't start adapter
	p.workingProxies = append(p.workingProxies, newProxy)
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
		// Check if already exists in working or reserve
		exists := false
		for _, existing := range p.workingProxies {
			if existing.String() == proxyStr {
				exists = true
				break
			}
		}
		if !exists {
			for _, existing := range p.reserveProxies {
				if existing.String() == proxyStr {
					exists = true
					break
				}
			}
		}
		if exists {
			continue
		}

		if proxy := ParseProxy(proxyStr); proxy != nil {
			// Decide where to put it
			if len(p.workingProxies) < p.minPoolSize {
				// Add to working (Ephemeral: Don't start adapter)
				p.workingProxies = append(p.workingProxies, *proxy)
			} else {
				// Add to reserve
				p.reserveProxies = append(p.reserveProxies, *proxy)
			}
			addedCount++
		}
	}
	if addedCount > 0 {
		log.Printf("Added %d new proxies. Working: %d, Reserve: %d", addedCount, len(p.workingProxies), len(p.reserveProxies))
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

func (p *MemoryProxyPool) UpdateProxiesFromIPs(baseLink string, ips []string) {
	if baseLink == "" || len(ips) == 0 {
		return
	}

	u, err := url.Parse(baseLink)
	if err != nil {
		log.Printf("Invalid base VLESS link: %v", err)
		return
	}

	// Shuffle IPs to get random selection
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(ips), func(i, j int) { ips[i], ips[j] = ips[j], ips[i] })

	// Limit to max 50 IPs to prevent starting too many Xray instances
	// We now support reserve proxies, so we can fetch more.
	maxIPs := 50
	if len(ips) > maxIPs {
		log.Printf("Limiting fetched IPs from %d to %d", len(ips), maxIPs)
		ips = ips[:maxIPs]
	}

	var newLinks []string
	for _, ip := range ips {
		// ip is "IP:Port"
		// We need to replace the host in the URL
		// u.Host contains "host:port" or just "host"
		// We replace it with the new ip:port
		
		// Clone the url (by value copy of struct, but url.URL has pointers? No, User is pointer)
		// Better to just modify a copy if possible, or reconstruct string.
		// url.URL struct fields are public.
		
		newU := *u // Shallow copy
		if u.User != nil {
			user := *u.User // Copy User info
			newU.User = &user
		}

		// Preserve original host as SNI/Host if not present
		q := newU.Query()
		originalHost := u.Hostname()
		
		if q.Get("sni") == "" {
			q.Set("sni", originalHost)
		}
		if q.Get("host") == "" {
			q.Set("host", originalHost)
		}
		newU.RawQuery = q.Encode()
		
		newU.Host = ip // Set new host:port
		
		// Update fragment (remark) to indicate it's an auto-fetched IP
		newU.Fragment = fmt.Sprintf("%s-auto", ip)
		
		newLinks = append(newLinks, newU.String())
	}

	log.Printf("Generated %d VLESS links from fetched IPs.", len(newLinks))
	p.AddProxies(newLinks)
	
	// Remove the base link to prevent using the blocked domain
	p.RemoveProxy(baseLink)
}

func (p *MemoryProxyPool) RemoveProxy(proxyStr string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	// Remove from workingProxies
	for i, px := range p.workingProxies {
		if px.Server == proxyStr || px.String() == proxyStr {
			p.workingProxies = append(p.workingProxies[:i], p.workingProxies[i+1:]...)
			// We continue to remove all instances if duplicates exist? 
			// Or just break? Let's break for now, assuming unique base link.
			// Actually, if we multiplexed, we might have multiple.
			// But Initialize multiplexes by appending duplicates.
			// So we should probably remove ALL instances of the base link.
			// To do that safely while iterating, we should filter.
			break 
		}
	}
	
	// Better removal for all instances:
	// Filter in place
	n := 0
	for _, px := range p.workingProxies {
		if px.Server != proxyStr && px.String() != proxyStr {
			p.workingProxies[n] = px
			n++
		}
	}
	p.workingProxies = p.workingProxies[:n]

	// If it's a VLESS adapter, close it
	// Check for exact match or key match
	// If multiplexed, keys are "vless://...#index"
	// We need to remove all adapters derived from this base link.
	
	// If it's a VLESS adapter, close any active instances using this config
	for socksAddr, vlessLink := range p.proxyToVless {
		if vlessLink == proxyStr || strings.HasPrefix(vlessLink, proxyStr) {
			if adapter, ok := p.activeAdapters[socksAddr]; ok {
				adapter.Close()
				delete(p.activeAdapters, socksAddr)
				log.Printf("Forced close of active adapter for removed proxy: %s", vlessLink[:30]+"...")
			}
			delete(p.proxyToVless, socksAddr)
			delete(p.busyProxies, vlessLink)
		}
	}
}

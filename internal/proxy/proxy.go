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

// ... Initialize, loadFromDisk, ParseProxy same ...

func (p *MemoryProxyPool) GetProxy() *Proxy {
	p.lock.Lock()
	defer p.lock.Unlock()

	// 1. Try to find a proxy from workingProxies that hasn't been used today
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	
	var available []*Proxy
	for i := range p.workingProxies {
		proxy := &p.workingProxies[i]
		host := getHost(proxy.Server)
		
		lastUsed, ok := p.usageMap[host]
		if !ok || lastUsed.Before(today) {
			available = append(available, proxy)
		}
	}
	
	if len(available) == 0 {
		return nil
	}
	
	// Pick random
	selected := available[rand.Intn(len(available))]
	
	// Mark as used
	host := getHost(selected.Server)
	p.usageMap[host] = now
	p.saveUsage()
	
	return selected
}

func getHost(server string) string {
	u, err := url.Parse(server)
	if err != nil {
		return server
	}
	return u.Hostname()
}

// ... MarkFailed, verifyBatch ...

func (p *MemoryProxyPool) CheckProxyFast(proxy Proxy) bool {
	return checkProxy(proxy, "https://httpbin.org/ip")
}

// ... AddProxies, SaveToDisk ...

func (p *MemoryProxyPool) Size() int {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return len(p.workingProxies)
}


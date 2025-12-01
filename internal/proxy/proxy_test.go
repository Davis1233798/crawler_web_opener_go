package proxy

import (
	"strings"
	"testing"
)

func TestRemoveProxy(t *testing.T) {
	pool := NewMemoryProxyPool("test_proxies.txt", 10)
	
	// Add some dummy proxies
	pool.workingProxies = []Proxy{
		{Server: "vless://base-link"},
		{Server: "http://other-proxy"},
		{Server: "vless://base-link"}, // Duplicate
	}
	
	// Remove base link
	pool.RemoveProxy("vless://base-link")
	
	// Check if removed
	for _, p := range pool.workingProxies {
		if p.Server == "vless://base-link" {
			t.Errorf("Failed to remove proxy: %s", p.Server)
		}
	}
	
	if len(pool.workingProxies) != 1 {
		t.Errorf("Expected 1 proxy, got %d", len(pool.workingProxies))
	}
	
	if pool.workingProxies[0].Server != "http://other-proxy" {
		t.Errorf("Unexpected proxy remaining: %s", pool.workingProxies[0].Server)
	}
}

func TestUpdateProxiesFromIPs_SNI(t *testing.T) {
	pool := NewMemoryProxyPool("test_proxies.txt", 10)
	
	// Base link without SNI
	baseLink := "vless://uuid@example.com:443?security=tls&type=ws"
	ips := []string{"1.2.3.4:443"}
	
	pool.UpdateProxiesFromIPs(baseLink, ips)
	
	if len(pool.workingProxies) != 1 {
		t.Fatalf("Expected 1 proxy, got %d", len(pool.workingProxies))
	}
	
	generatedLink := pool.workingProxies[0].Server
	
	// Check if IP is used
	if !strings.Contains(generatedLink, "1.2.3.4:443") {
		t.Errorf("Generated link does not contain IP: %s", generatedLink)
	}
	
	// Check if SNI is added
	if !strings.Contains(generatedLink, "sni=example.com") {
		t.Errorf("Generated link missing SNI: %s", generatedLink)
	}
	
	// Check if Host is added
	if !strings.Contains(generatedLink, "host=example.com") {
		t.Errorf("Generated link missing Host: %s", generatedLink)
	}
}

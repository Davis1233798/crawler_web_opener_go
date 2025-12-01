package proxy

import (
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

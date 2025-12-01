package proxy

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// FetchPreferredIPs fetches IPs from multiple API endpoints and returns a unique list of "IP:Port" strings.
func FetchPreferredIPs(apiURLs []string) ([]string, error) {
	uniqueIPs := make(map[string]bool)
	var ips []string

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	for _, url := range apiURLs {
		if url == "" {
			continue
		}
		log.Printf("Fetching IPs from: %s", url)
		resp, err := client.Get(url)
		if err != nil {
			log.Printf("Failed to fetch from %s: %v", url, err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read body from %s: %v", url, err)
			continue
		}

		lines := strings.Split(string(body), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// Handle CSV or simple IP:Port format
			// If CSV, assume IP is first column, Port is second (if exists), or IP:Port in first
			// Common format: IP,Port,Latency,... or IP:Port
			
			parts := strings.Split(line, ",")
			ipPart := strings.TrimSpace(parts[0])
			
			// Check if ipPart contains port
			if strings.Contains(ipPart, ":") {
				if !uniqueIPs[ipPart] {
					uniqueIPs[ipPart] = true
					ips = append(ips, ipPart)
				}
			} else if len(parts) >= 2 {
				// Try to find port in second column
				port := strings.TrimSpace(parts[1])
				// Simple check if port is numeric
				if port != "" {
					fullIP := fmt.Sprintf("%s:%s", ipPart, port)
					if !uniqueIPs[fullIP] {
						uniqueIPs[fullIP] = true
						ips = append(ips, fullIP)
					}
				}
			} else {
				// Default port 443 if just IP? Or skip?
				// Let's assume 443 if just IP
				fullIP := fmt.Sprintf("%s:443", ipPart)
				if !uniqueIPs[fullIP] {
					uniqueIPs[fullIP] = true
					ips = append(ips, fullIP)
				}
			}
		}
	}

	log.Printf("Fetched %d unique IPs.", len(ips))
	return ips, nil
}

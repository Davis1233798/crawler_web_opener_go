package proxy

import (
	"fmt"
	"io"
	"log"
	"net"
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
			// Also handle "  - 1.2.3.4  DESC" format (vfarid/cf-clean-ips)
			
			// Remove leading "- " if present
			line = strings.TrimPrefix(line, "- ")
			line = strings.TrimSpace(line)
			
			var ipPart, port string
			
			if strings.Contains(line, ",") {
				// CSV format
				parts := strings.Split(line, ",")
				ipPart = strings.TrimSpace(parts[0])
				if len(parts) >= 2 {
					port = strings.TrimSpace(parts[1])
				}
			} else {
				// Whitespace separated
				parts := strings.Fields(line)
				if len(parts) > 0 {
					ipPart = parts[0]
					// If there is a second part and it's numeric, it might be a port, but usually it's metadata in these lists
					// vfarid list: IP  CODE  DOMAIN  TIMESTAMP
					// So we assume port 443 unless IP contains it
				}
			}

			if ipPart == "" {
				continue
			}
			
			// Validate IP
			if net.ParseIP(ipPart) == nil {
				// log.Printf("Skipping invalid IP: %s", ipPart)
				continue
			}
			
			// Check if ipPart contains port
			if strings.Contains(ipPart, ":") {
				if !uniqueIPs[ipPart] {
					uniqueIPs[ipPart] = true
					ips = append(ips, ipPart)
				}
			} else {
				// If port was found in CSV (and is numeric), use it
				// Otherwise default to 443
				finalPort := "443"
				if port != "" {
					// Validate port is number
					// ... (omitted for brevity, just use if not empty)
					finalPort = port
				}
				
				fullIP := fmt.Sprintf("%s:%s", ipPart, finalPort)
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

package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type ProxyFetcher struct {
	Client *http.Client
}

func NewProxyFetcher() *ProxyFetcher {
	return &ProxyFetcher{
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

type GeonodeResponse struct {
	Data []struct {
		IP        string   `json:"ip"`
		Port      string   `json:"port"`
		Protocols []string `json:"protocols"`
	} `json:"data"`
}

func (f *ProxyFetcher) FetchGeonode(limit int) ([]string, error) {
	url := fmt.Sprintf("https://proxylist.geonode.com/api/proxy-list?limit=%d&page=1&sort_by=lastChecked&sort_type=desc&filterUpTime=90&anonymityLevel=elite&protocols=http,https,socks4,socks5", limit)

	resp, err := f.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("geonode api returned status: %d", resp.StatusCode)
	}

	var result GeonodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var proxies []string
	for _, item := range result.Data {
		protocol := "http"
		// Priority: socks5 > socks4 > http
		for _, p := range item.Protocols {
			if p == "socks5" {
				protocol = "socks5"
				break
			}
			if p == "socks4" && protocol != "socks5" {
				protocol = "socks4"
			}
		}
		proxies = append(proxies, fmt.Sprintf("%s://%s:%s", protocol, item.IP, item.Port))
	}
	return proxies, nil
}

func (f *ProxyFetcher) FetchProxyScrape() ([]string, error) {
	url := "https://api.proxyscrape.com/v4/free-proxy-list/get?request=display_proxies&proxy_format=protocolipport&format=text&anonymity=Elite&timeout=20000"

	resp, err := f.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("proxyscrape api returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(body), "\n")
	var proxies []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			proxies = append(proxies, line)
		}
	}
	return proxies, nil
}

func (f *ProxyFetcher) FetchAll(limit int) []string {
	var allProxies []string

	log.Println("Fetching proxies from Geonode...")
	geoProxies, err := f.FetchGeonode(limit)
	if err != nil {
		log.Printf("Error fetching from Geonode: %v", err)
	} else {
		log.Printf("Fetched %d proxies from Geonode", len(geoProxies))
		allProxies = append(allProxies, geoProxies...)
	}

	log.Println("Fetching proxies from ProxyScrape...")
	scrapeProxies, err := f.FetchProxyScrape()
	if err != nil {
		log.Printf("Error fetching from ProxyScrape: %v", err)
	} else {
		log.Printf("Fetched %d proxies from ProxyScrape", len(scrapeProxies))
		allProxies = append(allProxies, scrapeProxies...)
	}

	return unique(allProxies)
}

func unique(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

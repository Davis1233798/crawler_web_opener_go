package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
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

func (f *ProxyFetcher) FetchFreeProxyList() ([]string, error) {
	url := "https://free-proxy-list.net/"

	resp, err := f.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("free-proxy-list returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Simple regex to find IP:Port pairs in the table
	// The table structure is usually <td>IP</td><td>Port</td>
	re := regexp.MustCompile(`<td>(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})</td><td>(\d+)</td>`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	var proxies []string
	for _, match := range matches {
		if len(match) == 3 {
			// Defaulting to http, but free-proxy-list has https/socks sometimes in other columns
			// For simplicity, we assume http/https which most libraries handle auto-upgrade or just try
			proxies = append(proxies, fmt.Sprintf("http://%s:%s", match[1], match[2]))
		}
	}
	return proxies, nil
}

func (f *ProxyFetcher) FetchProxyListDownload() ([]string, error) {
	// API for http, https, socks4, socks5
	types := []string{"http", "https", "socks4", "socks5"}
	var allProxies []string

	for _, t := range types {
		url := fmt.Sprintf("https://www.proxy-list.download/api/v1/get?type=%s", t)
		resp, err := f.Client.Get(url)
		if err != nil {
			log.Printf("Error fetching ProxyListDownload type %s: %v", t, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		lines := strings.Split(string(body), "\r\n") // API returns \r\n
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				allProxies = append(allProxies, fmt.Sprintf("%s://%s", t, line))
			}
		}
	}
	return allProxies, nil
}

type PubProxyResponse struct {
	Data []struct {
		IPPort string `json:"ipPort"`
		Type   string `json:"type"` // http, socks4, socks5
	} `json:"data"`
}

func (f *ProxyFetcher) FetchPubProxy() ([]string, error) {
	// PubProxy limits to 50 per request for free
	url := "http://pubproxy.com/api/proxy?limit=20&format=json&type=http,socks4,socks5&level=elite"

	resp, err := f.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("pubproxy api returned status: %d", resp.StatusCode)
	}

	var result PubProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var proxies []string
	for _, item := range result.Data {
		proxies = append(proxies, fmt.Sprintf("%s://%s", item.Type, item.IPPort))
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

	log.Println("Fetching proxies from FreeProxyList...")
	freeProxies, err := f.FetchFreeProxyList()
	if err != nil {
		log.Printf("Error fetching from FreeProxyList: %v", err)
	} else {
		log.Printf("Fetched %d proxies from FreeProxyList", len(freeProxies))
		allProxies = append(allProxies, freeProxies...)
	}

	log.Println("Fetching proxies from ProxyListDownload...")
	dlProxies, err := f.FetchProxyListDownload()
	if err != nil {
		log.Printf("Error fetching from ProxyListDownload: %v", err)
	} else {
		log.Printf("Fetched %d proxies from ProxyListDownload", len(dlProxies))
		allProxies = append(allProxies, dlProxies...)
	}

	log.Println("Fetching proxies from PubProxy...")
	pubProxies, err := f.FetchPubProxy()
	if err != nil {
		log.Printf("Error fetching from PubProxy: %v", err)
	} else {
		log.Printf("Fetched %d proxies from PubProxy", len(pubProxies))
		allProxies = append(allProxies, pubProxies...)
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

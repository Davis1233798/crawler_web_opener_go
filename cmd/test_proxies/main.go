package main

import (
	"fmt"

	"github.com/Davis1233798/crawler-go/internal/proxy"
)

func main() {
	fetcher := proxy.NewProxyFetcher()
	proxies := fetcher.FetchAll(10)
	fmt.Printf("Total proxies fetched: %d\n", len(proxies))

	// Print a few to verify format
	if len(proxies) > 0 {
		fmt.Println("Sample proxies:")
		for i := 0; i < 5 && i < len(proxies); i++ {
			fmt.Println(proxies[i])
		}
	}
}

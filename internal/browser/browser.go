package browser

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Davis1233798/crawler-go/internal/proxy"
	"github.com/Davis1233798/crawler-go/pkg/fingerprint"
	"github.com/playwright-community/playwright-go"
)

type BrowserPool struct {
	pw       *playwright.Playwright
	browser  playwright.Browser
	headless bool
	mu       sync.RWMutex
}

func NewBrowserPool(headless bool) *BrowserPool {
	return &BrowserPool{
		headless: headless,
	}
}

func (bp *BrowserPool) Initialize() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.startBrowser()
}

func (bp *BrowserPool) startBrowser() error {
	var err error
	if bp.pw == nil {
		bp.pw, err = playwright.Run()
		if err != nil {
			return fmt.Errorf("could not start playwright: %v", err)
		}
	}

	if bp.browser != nil {
		if bp.browser.IsConnected() {
			return nil
		}
		bp.browser.Close()
	}

	log.Println("Launching new browser instance...")
	bp.browser, err = bp.pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(bp.headless),
		Proxy: &playwright.Proxy{
			Server: "http://per-context",
		},
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-dev-shm-usage", // Added to prevent crashes in Docker/low memory
		},
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %v", err)
	}
	return nil
}

func (bp *BrowserPool) CreateContext(p *proxy.Proxy) (playwright.BrowserContext, error) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	// Ensure browser is running
	if bp.browser == nil || !bp.browser.IsConnected() {
		log.Println("Browser disconnected, attempting restart...")
		if err := bp.startBrowser(); err != nil {
			return nil, fmt.Errorf("failed to restart browser: %v", err)
		}
	}

	fp := fingerprint.GetRandomFingerprint()

	cs := playwright.ColorScheme(fp.ColorScheme)
	opts := playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(fp.UserAgent),
		Viewport: &playwright.Size{
			Width:  fp.Viewport.Width,
			Height: fp.Viewport.Height,
		},
		Locale:            playwright.String(fp.Locale),
		TimezoneId:        playwright.String(fp.TimezoneID),
		ColorScheme:       &cs,
		DeviceScaleFactor: playwright.Float(fp.DeviceScaleFactor),
		IsMobile:          playwright.Bool(fp.IsMobile),
		HasTouch:          playwright.Bool(fp.HasTouch),
	}

	if p != nil {
		opts.Proxy = &playwright.Proxy{
			Server:   p.Server,
			Username: playwright.String(p.Username),
			Password: playwright.String(p.Password),
		}
	}

	context, err := bp.browser.NewContext(opts)
	if err != nil {
		return nil, err
	}

	// Inject stealth script
	script := fingerprint.GetStealthScript(fp)
	if err := context.AddInitScript(playwright.Script{Content: playwright.String(script)}); err != nil {
		context.Close()
		return nil, err
	}

	return context, nil
}

func (bp *BrowserPool) Shutdown() {
	if bp.browser != nil {
		bp.browser.Close()
	}
	if bp.pw != nil {
		bp.pw.Stop()
	}
}

type BrowserBot struct {
	pool *BrowserPool
}

func NewBrowserBot(pool *BrowserPool) *BrowserBot {
	return &BrowserBot{pool: pool}
}

func (bot *BrowserBot) Run(targets []string, p *proxy.Proxy, minDuration int) error {
	context, err := bot.pool.CreateContext(p)
	if err != nil {
		return err
	}
	defer context.Close()

	for _, url := range targets {
		page, err := context.NewPage()
		if err != nil {
			log.Printf("Failed to create page for %s: %v", url, err)
			// If browser is closed, abort the session so we can restart
			if strings.Contains(err.Error(), "closed") || strings.Contains(err.Error(), "crash") {
				return fmt.Errorf("browser crashed or closed: %w", err)
			}
			continue
		}

		// Auto-close popups
		page.On("popup", func(popup playwright.Page) {
			log.Println("Popup detected, closing...")
			popup.Close()
		})

		// Navigation
		log.Printf("Navigating to %s", url)
		if _, err := page.Goto(url, playwright.PageGotoOptions{
			Timeout:   playwright.Float(60000),
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); err != nil {
			log.Printf("Navigation failed for %s: %v", url, err)
			page.Close()
			continue
		}

		// Watch videos
		if err := bot.watchVideos(page, p); err != nil {
			log.Printf("Video watching failed for %s: %v", url, err)
		}

		page.Close()
	}

	return nil
}

func (bot *BrowserBot) watchVideos(page playwright.Page, p *proxy.Proxy) error {
	// Wait for potential video load - try to wait for selector first
	log.Println("Waiting for video element...")
	tryWait := func() {
		page.WaitForSelector("video", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(10000),
		})
	}
	tryWait()

	// Find all video tags (including in frames)
	var videos []playwright.Locator

	// Main frame videos
	mainVideos, err := page.Locator("video").All()
	if err == nil {
		videos = append(videos, mainVideos...)
	}

	// Check frames
	for _, frame := range page.Frames() {
		frameVideos, err := frame.Locator("video").All()
		if err == nil {
			videos = append(videos, frameVideos...)
		}
	}

	if len(videos) == 0 {
		title, _ := page.Title()
		content, _ := page.Content()
		log.Printf("No videos found on page. Title: '%s'", title)
		if len(content) > 500 {
			log.Printf("Page content preview: %s...", content[:500])
		}
		// Take screenshot for debug
		if _, err := page.Screenshot(playwright.PageScreenshotOptions{
			Path: playwright.String("debug_no_video.png"),
		}); err != nil {
			log.Printf("Failed to take screenshot: %v", err)
		}
		
		time.Sleep(2 * time.Second)
		return nil
	}

	log.Printf("Found %d videos", len(videos))

	for i, video := range videos {
		log.Printf("Watching video %d/%d", i+1, len(videos))

		// Ensure visible
		video.ScrollIntoViewIfNeeded()
		time.Sleep(1 * time.Second)

		// Get Video Source for downloading
		src, _ := video.GetAttribute("src")
		if src == "" {
			// Try source tags
			if sources, err := video.Locator("source").All(); err == nil && len(sources) > 0 {
				src, _ = sources[0].GetAttribute("src")
			}
		}

		// Start Download in background
		var downloadWg sync.WaitGroup
		if src != "" && !strings.HasPrefix(src, "blob:") {
			downloadWg.Add(1)
			go func(videoUrl string) {
				defer downloadWg.Done()
				bot.downloadAndCleanup(page, videoUrl, p)
			}(src)
		} else {
			log.Println("Video source not found or is blob, skipping download")
		}

		// Play
		log.Printf("Attempting to play video %d...", i+1)
		
		// Ad-Busting / Click-Through Logic
		// Many streaming sites require multiple clicks to clear invisible overlays/ads before the video plays.
		maxClicks := 10
		isPlaying := false
		
		for attempt := 0; attempt < maxClicks; attempt++ {
			// Check if playing
			paused, err := video.Evaluate("v => v.paused", nil)
			if err == nil && !paused.(bool) {
				isPlaying = true
				log.Println("Video is playing!")
				break
			}

			log.Printf("Click attempt %d/%d to start video...", attempt+1, maxClicks)

			// Try different click targets
			// 1. Try the specific JW Player display icon if it exists (common in the user's provided HTML)
			jwDisplay := page.Locator(".jw-display-icon-display").First()
			if visible, _ := jwDisplay.IsVisible(); visible {
				jwDisplay.Click(playwright.LocatorClickOptions{Timeout: playwright.Float(1000)})
			} else {
				// Fallback to clicking the video element itself
				video.Click(playwright.LocatorClickOptions{Timeout: playwright.Float(1000)})
			}

			// Also try forcing play via JS
			video.Evaluate("v => v.play()", nil)

			// Wait a bit for ads to trigger or video to start
			time.Sleep(2 * time.Second)
			
			// If a new tab/popup opened, the main page might have lost focus or paused.
			// The popup handler in Run() closes them, but we need to ensure we keep interacting.
		}

		if !isPlaying {
			log.Printf("Warning: Could not verify if video %d started playing after %d attempts", i+1, maxClicks)
		}

		// Monitor video
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		lastTime := 0.0
		stuckCount := 0

		for {
			<-ticker.C
			if page.IsClosed() {
				// Page closed unexpectedly (maybe popup handler closed main page?)
				// But we are in a loop of targets.
				return fmt.Errorf("page closed")
			}

			// Check if ended
			ended, err := video.Evaluate("v => v.ended", nil)
			if err != nil {
				log.Printf("Error checking video status: %v", err)
				break
			}
			if ended.(bool) {
				log.Printf("Video %d ended", i+1)
				break
			}

			// Check if paused and resume
			paused, _ := video.Evaluate("v => v.paused", nil)
			if paused.(bool) {
				log.Println("Video paused, resuming...")
				video.Evaluate("v => v.play()", nil)
			}

			// Check progress to detect stuck videos
			currentTime, _ := video.Evaluate("v => v.currentTime", nil)
			if ct, ok := currentTime.(float64); ok {
				if ct == lastTime {
					stuckCount++
					if stuckCount > 15 { // Stuck for 30 seconds
						log.Println("Video stuck, skipping...")
						break
					}
				} else {
					lastTime = ct
					stuckCount = 0
				}
			}
		}

		// Wait for download to finish before moving to next video
		downloadWg.Wait()
	}
	return nil
}

func (bot *BrowserBot) downloadAndCleanup(page playwright.Page, videoUrl string, p *proxy.Proxy) {
	log.Printf("Starting download for: %s", videoUrl)

	// Prepare Client
	client := &http.Client{
		Timeout: 10 * time.Minute, // Allow long downloads
	}

	if p != nil {
		proxyUrl, err := url.Parse(p.ToURL())
		if err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
			}
		}
	}

	req, err := http.NewRequest("GET", videoUrl, nil)
	if err != nil {
		log.Printf("Failed to create download request: %v", err)
		return
	}

	// Copy headers from browser context
	ua, _ := page.Evaluate("navigator.userAgent", nil)
	if uaStr, ok := ua.(string); ok {
		req.Header.Set("User-Agent", uaStr)
	}
	req.Header.Set("Referer", page.URL())

	// Copy cookies
	cookies, err := page.Context().Cookies()
	if err == nil {
		for _, c := range cookies {
			req.AddCookie(&http.Cookie{
				Name:  c.Name,
				Value: c.Value,
			})
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Download failed: %v", err)
		return
	}
	defer resp.Body.Close()

	// Create temp file
	tmpFile, err := os.CreateTemp("", "video-*.mp4")
	if err != nil {
		log.Printf("Failed to create temp file: %v", err)
		return
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
		log.Printf("Deleted temp file: %s", tmpPath)
	}()

	// Download
	n, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		log.Printf("Error during download: %v", err)
		return
	}

	log.Printf("Download completed: %d bytes. Cleaning up...", n)
}

func (bot *BrowserBot) humanMouseMove(page playwright.Page) {
	// Simplified random movement
	size := page.ViewportSize()
	if size == nil {
		return
	}
	x := rand.Intn(size.Width)
	y := rand.Intn(size.Height)
	page.Mouse().Move(float64(x), float64(y), playwright.MouseMoveOptions{
		Steps: playwright.Int(rand.Intn(20) + 10),
	})
}

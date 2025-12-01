package browser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Davis1233798/crawler-go/internal/config"
	"github.com/Davis1233798/crawler-go/internal/proxy"
	"github.com/Davis1233798/crawler-go/pkg/fingerprint"
	"github.com/playwright-community/playwright-go"
)

// BrowserManager handles the Playwright driver
type BrowserManager struct {
	pw       *playwright.Playwright
	headless bool
	mu       sync.Mutex
}

func NewBrowserManager(headless bool) *BrowserManager {
	return &BrowserManager{
		headless: headless,
	}
}

func (bm *BrowserManager) Initialize() error {
	var err error
	bm.pw, err = playwright.Run()
	if err != nil {
		return fmt.Errorf("could not start playwright: %v", err)
	}
	log.Printf("Playwright driver started (Headless config: %v)", bm.headless)
	return nil
}

func (bm *BrowserManager) Shutdown() {
	if bm.pw != nil {
		bm.pw.Stop()
	}
}

func (bm *BrowserManager) LaunchBrowser(p *proxy.Proxy) (playwright.Browser, error) {
	// Launch a new browser instance
	// We configure proxy here if it's a global proxy, but we use per-context proxy usually.
	// However, to ensure "complete process closure", we launch a browser.
	
	opts := playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(bm.headless),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-infobars",
		},
	}

	// If we wanted to use proxy at browser level:
	// opts.Proxy = ...
	// But we stick to context-level proxy for flexibility, or we can do it here.
	// Since we launch ONE browser for ONE proxy session (as per new requirement), 
	// we CAN set it here, but context level is safer for auth.
	// Let's stick to context level, but launch the browser.

	browser, err := bm.pw.Chromium.Launch(opts)
	if err != nil {
		return nil, fmt.Errorf("could not launch browser: %v", err)
	}
	return browser, nil
}

type BrowserBot struct {
	manager    *BrowserManager
	clickCount int
	mu         sync.Mutex
}

func NewBrowserBot(manager *BrowserManager) *BrowserBot {
	return &BrowserBot{manager: manager}
}

// RunBatch launches a FRESH browser, runs the batch, and closes it.
func (bot *BrowserBot) RunBatch(urls []string, p *proxy.Proxy, minDuration int) error {
	if len(urls) == 0 {
		return nil
	}

	// 1. Check IP (Optional, but good for verification)
	currentIP := "Unknown"
	if p != nil {
		proxyURL, err := url.Parse(p.ToURL())
		if err == nil {
			client := &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
				},
				Timeout: 10 * time.Second,
			}
			resp, err := client.Get("https://api.ipify.org")
			if err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				currentIP = string(body)
				log.Printf("🔌 Connected via Proxy IP: %s", currentIP)
			} else {
				log.Printf("⚠️ Failed to check IP via proxy: %v", err)
			}
		}
	}

	// Send Discord Notification
	bot.sendDiscordNotification(currentIP)
	defer bot.sendDiscordNotification(currentIP) // Final report

	// 2. Launch Fresh Browser
	browserInstance, err := bot.manager.LaunchBrowser(p)
	if err != nil {
		return err
	}
	// Ensure browser is closed at the end of this batch
	defer func() {
		log.Println("🛑 Closing browser process...")
		browserInstance.Close()
	}()

	// 3. Create Context with Fingerprint
	fp := fingerprint.GetRandomFingerprint()
	log.Printf("🕵️ Fingerprint: %s | Res: %dx%d | OS: %s", fp.UserAgent[:30]+"...", fp.Screen.Width, fp.Screen.Height, fp.Extra.WebGL.Vendor)

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

	context, err := browserInstance.NewContext(opts)
	if err != nil {
		return err
	}
	defer context.Close()

	// Inject stealth script
	script := fingerprint.GetStealthScript(fp)
	if err := context.AddInitScript(playwright.Script{Content: playwright.String(script)}); err != nil {
		return err
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(urls))

	log.Printf("Opening %d tabs in parallel...", len(urls))

	// Reset click count
	bot.mu.Lock()
	bot.clickCount = 0
	bot.mu.Unlock()

	for _, u := range urls {
		wg.Add(1)
		go func(targetURL string) {
			defer wg.Done()

			page, err := context.NewPage()
			if err != nil {
				log.Printf("Failed to create page for %s: %v", targetURL, err)
				errChan <- err
				return
			}
			
			// Handle popups
			page.On("popup", func(popup playwright.Page) {
				log.Println("⚠️ Popup detected, closing it.")
				popup.Close()
			})

			log.Printf("Navigating to %s", targetURL)
			if _, err := page.Goto(targetURL, playwright.PageGotoOptions{
				Timeout:   playwright.Float(30000),                   // Reduced to 30s
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			}); err != nil {
				log.Printf("Navigation failed for %s: %v", targetURL, err)
				errChan <- err
				return
			}

			log.Printf("⏳ Activity started for %s (%ds)", targetURL, minDuration)
			bot.simulateActivity(page, minDuration)
			log.Printf("✅ Activity finished for %s", targetURL)
		}(u)
	}

	wg.Wait()
	close(errChan)
	log.Println("Batch finished, closing browser...")

	return nil
}

func (bot *BrowserBot) simulateActivity(page playwright.Page, durationSeconds int) {
	startTime := time.Now()
	for time.Since(startTime).Seconds() < float64(durationSeconds) {
		if page.IsClosed() {
			break
		}

		action := rand.Intn(4)
		switch action {
		case 0: // Scroll
			scrollAmount := rand.Intn(400) + 100
			page.Evaluate(fmt.Sprintf("window.scrollBy(0, %d)", scrollAmount))
		case 1: // Mouse Move
			bot.humanMouseMove(page)
		case 2: // Pause
			// Reduced pause for higher frequency
			time.Sleep(time.Duration(rand.Intn(1000)+500) * time.Millisecond)
		case 3: // Click random element (New)
			bot.clickRandomElement(page)
		}

		// Reduced interval between actions
		time.Sleep(time.Duration(rand.Intn(500)+200) * time.Millisecond)
	}
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
		Steps: playwright.Int(rand.Intn(5) + 5), // Faster movement
	})
}

func (bot *BrowserBot) clickRandomElement(page playwright.Page) {
	// Try to find clickable elements and click one
	// This is a simple heuristic
	handle, err := page.QuerySelector("a, button, input[type='submit']")
	if err == nil && handle != nil {
		// Move to it first
		box, err := handle.BoundingBox()
		if err == nil && box != nil {
			page.Mouse().Move(box.X+box.Width/2, box.Y+box.Height/2, playwright.MouseMoveOptions{
				Steps: playwright.Int(10),
			})
			// Click
			if err := handle.Click(playwright.ElementHandleClickOptions{
				Delay: playwright.Float(float64(rand.Intn(100) + 50)),
			}); err == nil {
				bot.mu.Lock()
				bot.clickCount++
				bot.mu.Unlock()
			}
		}
	}
}

func (bot *BrowserBot) sendDiscordNotification(ip string) {
	cfg := config.GetConfig()
	if cfg.DiscordWebhookURL == "" {
		return
	}

	bot.mu.Lock()
	count := bot.clickCount
	bot.mu.Unlock()

	message := map[string]interface{}{
		"content": fmt.Sprintf("🤖 **Crawler Report**\n🌐 IP: `%s`\n👆 Total Clicks: `%d`", ip, count),
	}

	jsonData, _ := json.Marshal(message)
	resp, err := http.Post(cfg.DiscordWebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to send Discord notification: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Println("✅ Discord notification sent.")
}

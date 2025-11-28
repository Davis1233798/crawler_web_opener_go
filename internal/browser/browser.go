package browser

import (
	"fmt"
	"log"
	"math/rand"
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
}

func NewBrowserPool(headless bool) *BrowserPool {
	return &BrowserPool{
		headless: headless,
	}
}

func (bp *BrowserPool) Initialize() error {
	var err error
	bp.pw, err = playwright.Run()
	if err != nil {
		return fmt.Errorf("could not start playwright: %v", err)
	}

	bp.browser, err = bp.pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(bp.headless),
		Proxy: &playwright.Proxy{
			Server: "http://per-context",
		},
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--no-sandbox",
			"--disable-setuid-sandbox",
		},
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %v", err)
	}
	return nil
}

func (bp *BrowserPool) CreateContext(p *proxy.Proxy) (playwright.BrowserContext, error) {
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

func (bot *BrowserBot) Run(url string, p *proxy.Proxy, minDuration int) error {
	context, err := bot.pool.CreateContext(p)
	if err != nil {
		return err
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		return err
	}

	// Navigation
	log.Printf("Navigating to %s", url)
	if _, err := page.Goto(url, playwright.PageGotoOptions{
		Timeout:   playwright.Float(30000),
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		return fmt.Errorf("navigation failed: %v", err)
	}

	// Simulate activity
	bot.simulateActivity(page, minDuration)

	return nil
}

func (bot *BrowserBot) RunBatch(urls []string, p *proxy.Proxy, minDuration int) error {
	if len(urls) == 0 {
		return nil
	}

	context, err := bot.pool.CreateContext(p)
	if err != nil {
		return err
	}
	defer context.Close()

	var wg sync.WaitGroup
	errChan := make(chan error, len(urls))

	log.Printf("Opening %d tabs in parallel...", len(urls))

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
			// We don't defer page.Close() here because context.Close() will handle it,
			// and we want them open simultaneously.

			log.Printf("Navigating to %s", targetURL)
			if _, err := page.Goto(targetURL, playwright.PageGotoOptions{
				Timeout:   playwright.Float(60000),                   // Increased timeout for batch
				WaitUntil: playwright.WaitUntilStateDomcontentloaded, // Faster than networkidle for batch
			}); err != nil {
				log.Printf("Navigation failed for %s: %v", targetURL, err)
				errChan <- err
				return
			}

			bot.simulateActivity(page, minDuration)
		}(u)
	}

	wg.Wait()
	close(errChan)

	// Collect errors if needed, but for now we just log them in the loop
	// If all failed, we might want to return an error, but partial success is okay.
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
			time.Sleep(time.Duration(rand.Intn(3)+1) * time.Second)
		case 3: // Click random element (New)
			bot.clickRandomElement(page)
		}

		time.Sleep(time.Duration(rand.Intn(2000)+500) * time.Millisecond)
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
		Steps: playwright.Int(rand.Intn(20) + 10),
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
			handle.Click(playwright.ElementHandleClickOptions{
				Delay: playwright.Float(float64(rand.Intn(100) + 50)),
			})
		}
	}
}

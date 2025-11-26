package browser

import (
	"fmt"
	"log"
	"math/rand"
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
	startTime := time.Now()
	for time.Since(startTime).Seconds() < float64(minDuration) {
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
		case 3: // Select text (simplified)
			// ... implementation omitted for brevity, can add later
		}

		time.Sleep(time.Duration(rand.Intn(2000)+500) * time.Millisecond)
	}

	return nil
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

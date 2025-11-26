package fingerprint

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type Viewport struct {
	Width  int
	Height int
}

type Screen struct {
	Width  int
	Height int
}

type WebGLConfig struct {
	Vendor   string
	Renderer string
}

type AudioConfig struct {
	SampleRate   int
	ChannelCount int
	BufferSize   int
}

type Fingerprint struct {
	UserAgent         string
	Viewport          Viewport
	Locale            string
	TimezoneID        string
	ColorScheme       string
	DeviceScaleFactor float64
	IsMobile          bool
	HasTouch          bool
	Screen            Screen
	Extra             ExtraConfig
}

type ExtraConfig struct {
	Fonts               []string
	Audio               AudioConfig
	WebGL               WebGLConfig
	HardwareConcurrency int
	DeviceMemory        int
}

var (
	UserAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
	}

	FingerprintTemplates = []struct {
		Viewport    Viewport
		Locale      string
		TimezoneID  string
		ColorScheme string
	}{
		{Viewport{1920, 1080}, "en-US", "America/New_York", "dark"},
		{Viewport{1366, 768}, "en-US", "America/Los_Angeles", "light"},
		{Viewport{1536, 864}, "en-GB", "Europe/London", "dark"},
		{Viewport{1440, 900}, "zh-TW", "Asia/Taipei", "light"},
		{Viewport{1280, 720}, "zh-CN", "Asia/Shanghai", "dark"},
		{Viewport{2560, 1440}, "en-US", "America/Chicago", "light"},
		{Viewport{1680, 1050}, "ja-JP", "Asia/Tokyo", "dark"},
		{Viewport{1600, 900}, "ko-KR", "Asia/Seoul", "light"},
	}

	WebGLConfigs = []WebGLConfig{
		{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA GeForce RTX 3060 Direct3D11 vs_5_0 ps_5_0)"},
		{"Google Inc. (Intel)", "ANGLE (Intel(R) UHD Graphics 630 Direct3D11 vs_5_0 ps_5_0)"},
		{"Google Inc. (AMD)", "ANGLE (AMD Radeon RX 6700 XT Direct3D11 vs_5_0 ps_5_0)"},
		{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA GeForce GTX 1660 Ti Direct3D11 vs_5_0 ps_5_0)"},
		{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA GeForce RTX 4070 Direct3D11 vs_5_0 ps_5_0)"},
		{"Google Inc. (AMD)", "ANGLE (AMD Radeon RX 7800 XT Direct3D11 vs_5_0 ps_5_0)"},
	}

	FontSets = [][]string{
		{"Arial", "Calibri", "Cambria", "Consolas", "Times New Roman", "Segoe UI", "Verdana"},
		{"Arial", "Helvetica", "San Francisco", "Monaco", "Courier New", "Times", "Verdana"},
		{"DejaVu Sans", "Liberation Sans", "Ubuntu", "Noto Sans", "Arial", "FreeSans"},
	}

	AudioConfigs = []AudioConfig{
		{44100, 2, 4096},
		{48000, 2, 8192},
		{44100, 1, 2048},
	}
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GetRandomFingerprint() Fingerprint {
	template := FingerprintTemplates[rand.Intn(len(FingerprintTemplates))]

	viewportWidth := template.Viewport.Width
	viewportHeight := template.Viewport.Height
	screenWidth := viewportWidth + rand.Intn(200)
	screenHeight := viewportHeight + rand.Intn(200) + 100

	deviceScaleFactors := []float64{1, 1.25, 1.5, 2}

	return Fingerprint{
		UserAgent:         UserAgents[rand.Intn(len(UserAgents))],
		Viewport:          template.Viewport,
		Locale:            template.Locale,
		TimezoneID:        template.TimezoneID,
		ColorScheme:       template.ColorScheme,
		DeviceScaleFactor: deviceScaleFactors[rand.Intn(len(deviceScaleFactors))],
		IsMobile:          false,
		HasTouch:          false,
		Screen:            Screen{Width: screenWidth, Height: screenHeight},
		Extra: ExtraConfig{
			Fonts:               FontSets[rand.Intn(len(FontSets))],
			Audio:               AudioConfigs[rand.Intn(len(AudioConfigs))],
			WebGL:               WebGLConfigs[rand.Intn(len(WebGLConfigs))],
			HardwareConcurrency: rand.Intn(13) + 4, // 4-16
			DeviceMemory:        []int{4, 8, 16, 32}[rand.Intn(4)],
		},
	}
}

func GetStealthScript(fp Fingerprint) string {
	audioNoise := 0.0001 + rand.Float64()*(0.0005-0.0001)

	var fontsQuoted []string
	for _, f := range fp.Extra.Fonts {
		fontsQuoted = append(fontsQuoted, fmt.Sprintf("'%s'", f))
	}
	fontsJs := strings.Join(fontsQuoted, ", ")

	script := fmt.Sprintf(`
    (function() {
        'use strict';
        
        // ===== 基礎反檢測 =====
        Object.defineProperty(navigator, 'webdriver', {
            get: () => undefined,
            configurable: true
        });
        
        delete navigator.__webdriver_evaluate;
        delete navigator.__driver_evaluate;
        delete navigator.__webdriver_script_function;
        delete navigator.__webdriver_script_func;
        delete navigator.__webdriver_script_fn;
        delete navigator.__fxdriver_evaluate;
        delete navigator.__driver_unwrapped;
        delete navigator.__webdriver_unwrapped;
        
        // ===== 硬體資訊隨機化 =====
        Object.defineProperty(navigator, 'hardwareConcurrency', {
            get: () => %d,
            configurable: true
        });
        
        Object.defineProperty(navigator, 'deviceMemory', {
            get: () => %d,
            configurable: true
        });
        
        // ===== WebGL 指紋偽裝 =====
        const getParameter = WebGLRenderingContext.prototype.getParameter;
        WebGLRenderingContext.prototype.getParameter = function(parameter) {
            if (parameter === 37445) return '%s';
            if (parameter === 37446) return '%s';
            return getParameter.apply(this, arguments);
        };
        
        if (window.WebGL2RenderingContext) {
            const getParameter2 = WebGL2RenderingContext.prototype.getParameter;
            WebGL2RenderingContext.prototype.getParameter = function(parameter) {
                if (parameter === 37445) return '%s';
                if (parameter === 37446) return '%s';
                return getParameter2.apply(this, arguments);
            };
        }
        
        // ===== Canvas 指紋隨機化 =====
        const shift = {
            r: Math.floor(Math.random() * 10) - 5,
            g: Math.floor(Math.random() * 10) - 5,
            b: Math.floor(Math.random() * 10) - 5,
            a: Math.floor(Math.random() * 10) - 5
        };
        
        const originalToDataURL = HTMLCanvasElement.prototype.toDataURL;
        HTMLCanvasElement.prototype.toDataURL = function(type) {
            const context = this.getContext('2d');
            if (context) {
                const imageData = context.getImageData(0, 0, this.width, this.height);
                for (let i = 0; i < imageData.data.length; i += 4) {
                    imageData.data[i] = imageData.data[i] + shift.r;
                    imageData.data[i + 1] = imageData.data[i + 1] + shift.g;
                    imageData.data[i + 2] = imageData.data[i + 2] + shift.b;
                    imageData.data[i + 3] = imageData.data[i + 3] + shift.a;
                }
                context.putImageData(imageData, 0, 0);
            }
            return originalToDataURL.apply(this, arguments);
        };
        
        // ===== 音訊上下文指紋隨機化 =====
        const AudioContext = window.AudioContext || window.webkitAudioContext;
        if (AudioContext) {
            const OriginalAnalyser = AudioContext.prototype.createAnalyser;
            AudioContext.prototype.createAnalyser = function() {
                const analyser = OriginalAnalyser.call(this);
                const originalGetFloatFrequencyData = analyser.getFloatFrequencyData;
                analyser.getFloatFrequencyData = function(array) {
                    originalGetFloatFrequencyData.call(this, array);
                    for (let i = 0; i < array.length; i++) {
                        array[i] += %f * (Math.random() - 0.5);
                    }
                };
                return analyser;
            };
        }
        
        // ===== 字體指紋偽裝 =====
        const availableFonts = [%s];
        Object.defineProperty(document, 'fonts', {
            get: () => ({
                check: (font) => availableFonts.some(f => font.includes(f)),
                ready: Promise.resolve(),
                size: availableFonts.length
            })
        });
        
        // ===== WebRTC IP 洩漏防護 =====
        const originalRTCPeerConnection = window.RTCPeerConnection;
        if (originalRTCPeerConnection) {
            window.RTCPeerConnection = function(config) {
                if (config && config.iceServers) {
                    config.iceServers = config.iceServers.filter(server => {
                        return !server.urls || !server.urls.toString().includes('stun');
                    });
                }
                return new originalRTCPeerConnection(config);
            };
        }

         // ===== Battery API 隨機化 =====
        if (navigator.getBattery) {
            const originalGetBattery = navigator.getBattery;
            navigator.getBattery = function() {
                return originalGetBattery.call(this).then(battery => {
                    Object.defineProperties(battery, {
                        charging: { get: () => Math.random() > 0.5 },
                        level: { get: () => 0.5 + Math.random() * 0.5 },
                        chargingTime: { get: () => Infinity },
                        dischargingTime: { get: () => Math.random() * 10000 + 5000 }
                    });
                    return battery;
                });
            };
        }

    })();
    `,
		fp.Extra.HardwareConcurrency,
		fp.Extra.DeviceMemory,
		fp.Extra.WebGL.Vendor, fp.Extra.WebGL.Renderer,
		fp.Extra.WebGL.Vendor, fp.Extra.WebGL.Renderer,
		audioNoise,
		fontsJs,
	)

	return script
}

package config

import (
	"bufio"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Threads         int
	Duration        int
	Headless        bool
	BrowserPoolSize int
	MetricsPort       int
	DiscordWebhookURL string
	Targets           []string
}

var (
	instance *Config
	once     sync.Once
)

// GetConfig returns the singleton configuration instance
func GetConfig() *Config {
	once.Do(func() {
		instance = &Config{}
		instance.load()
	})
	return instance
}

func (c *Config) load() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	c.Threads = getEnvAsInt("THREADS", 10)
	c.Duration = getEnvAsInt("DURATION", 30)
	c.Headless = getEnvAsBool("HEADLESS", false)
	c.BrowserPoolSize = getEnvAsInt("BROWSER_POOL_SIZE", 5)
	c.MetricsPort = getEnvAsInt("METRICS_PORT", 8000)
	c.DiscordWebhookURL = getEnv("DISCORD_WEBHOOK_URL", "")

	c.loadTargets()
}

func (c *Config) loadTargets() {
	file, err := os.Open("target_site.txt")
	if err != nil {
		log.Printf("Error opening target_site.txt: %v", err)
		c.Targets = []string{}
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			c.Targets = append(c.Targets, line)
		}
	}
	log.Printf("Loaded %d target sites.", len(c.Targets))
}

func (c *Config) GetRandomTarget() string {
	if len(c.Targets) == 0 {
		return ""
	}
	rand.Seed(time.Now().UnixNano())
	return c.Targets[rand.Intn(len(c.Targets))]
}

// Helper functions
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valueStr := getEnv(key, "")
	if val, err := strconv.ParseBool(valueStr); err == nil {
		return val
	}
	return defaultVal
}

package vless

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
)

type VlessConfig struct {
	UUID     string
	Host     string
	Port     int
	Network  string
	Security string
	Path     string
	Sni      string
}

type Manager struct {
	ConfigPath string
	XrayPath   string
	LocalPort  int
	cmd        *exec.Cmd
}

func NewManager(xrayPath string, localPort int) *Manager {
	return &Manager{
		ConfigPath: "config.json",
		XrayPath:   xrayPath,
		LocalPort:  localPort,
	}
}

func (m *Manager) ParseVless(vlessStr string) (*VlessConfig, error) {
	u, err := url.Parse(vlessStr)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "vless" {
		return nil, fmt.Errorf("invalid scheme: %s", u.Scheme)
	}

	uuid := u.User.Username()
	host := u.Hostname()
	port, _ := strconv.Atoi(u.Port())
	q := u.Query()

	return &VlessConfig{
		UUID:     uuid,
		Host:     host,
		Port:     port,
		Network:  q.Get("type"), // often 'type' or 'network'
		Security: q.Get("security"),
		Path:     q.Get("path"),
		Sni:      q.Get("sni"),
	}, nil
}

func (m *Manager) GenerateConfig(vless *VlessConfig) error {
	// Basic stream settings
	streamSettings := map[string]interface{}{
		"network": vless.Network,
	}

	if vless.Security == "tls" {
		streamSettings["security"] = "tls"
		streamSettings["tlsSettings"] = map[string]interface{}{
			"serverName": vless.Sni,
		}
	}

	if vless.Network == "ws" {
		streamSettings["wsSettings"] = map[string]interface{}{
			"path": vless.Path,
		}
	}

	config := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "warning",
		},
		"inbounds": []map[string]interface{}{
			{
				"port":     m.LocalPort,
				"protocol": "socks",
				"settings": map[string]interface{}{
					"auth": "noauth",
				},
				"sniffing": map[string]interface{}{
					"enabled":      true,
					"destOverride": []string{"http", "tls"},
				},
			},
		},
		"outbounds": []map[string]interface{}{
			{
				"protocol": "vless",
				"settings": map[string]interface{}{
					"vnext": []map[string]interface{}{
						{
							"address": vless.Host,
							"port":    vless.Port,
							"users": []map[string]interface{}{
								{
									"id":         vless.UUID,
									"encryption": "none",
								},
							},
						},
					},
				},
				"streamSettings": streamSettings,
			},
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.ConfigPath, data, 0644)
}

func (m *Manager) Start() error {
	// Check if xray exists
	if _, err := exec.LookPath(m.XrayPath); err != nil {
		// Try current directory
		if _, err := os.Stat(m.XrayPath); os.IsNotExist(err) {
			return fmt.Errorf("xray binary not found at %s or in PATH", m.XrayPath)
		}
	}

	m.cmd = exec.Command(m.XrayPath, "-c", m.ConfigPath)
	// Redirect stdout/stderr to files or ignore to avoid clutter
	outfile, _ := os.Create("xray.log")
	m.cmd.Stdout = outfile
	m.cmd.Stderr = outfile

	return m.cmd.Start()
}

func (m *Manager) Stop() {
	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
	}
}

package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync"

	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"
	_ "github.com/xtls/xray-core/main/distro/all" // Import all necessary modules
)

// VLESSAdapter manages a local Xray instance that acts as a SOCKS5 adapter for a VLESS proxy
type VLESSAdapter struct {
	Instance   *core.Instance
	LocalPort  int
	VLESSLink  string
	CancelFunc context.CancelFunc
	mu         sync.Mutex
}

// StartVLESSAdapter starts a local Xray instance mapping a random SOCKS5 port to the VLESS target
func StartVLESSAdapter(vlessLink string) (*VLESSAdapter, error) {
	// Parse VLESS link
	config, err := parseVLESSLink(vlessLink)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vless link: %v", err)
	}

	// Find a free port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to find free port: %v", err)
	}
	localPort := l.Addr().(*net.TCPAddr).Port
	l.Close()

	// Build Xray config
	xrayConfig, err := buildXrayConfig(localPort, config)
	if err != nil {
		return nil, fmt.Errorf("failed to build xray config: %v", err)
	}

	// Create Xray instance
	instance, err := core.New(xrayConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create xray instance: %v", err)
	}

	if err := instance.Start(); err != nil {
		return nil, fmt.Errorf("failed to start xray instance: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	adapter := &VLESSAdapter{
		Instance:   instance,
		LocalPort:  localPort,
		VLESSLink:  vlessLink,
		CancelFunc: cancel,
	}

	// Monitor instance (optional)
	go func() {
		<-ctx.Done()
		instance.Close()
	}()

	return adapter, nil
}

func (a *VLESSAdapter) SocksAddr() string {
	return fmt.Sprintf("127.0.0.1:%d", a.LocalPort)
}

func (a *VLESSAdapter) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.CancelFunc != nil {
		a.CancelFunc()
	}
}

type VLESSConfig struct {
	UUID     string
	Address  string
	Port     uint32
	Flow     string
	Security string
	SNI      string
	FP       string
	Type     string
	Path     string
	Host     string
	PBK      string
	SID      string
	SpiderX  string
}

func parseVLESSLink(link string) (*VLESSConfig, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "vless" {
		return nil, fmt.Errorf("invalid scheme: %s", u.Scheme)
	}

	uuid := u.User.Username()
	host := u.Hostname()
	portStr := u.Port()
	portInt, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %s", portStr)
	}

	q := u.Query()

	return &VLESSConfig{
		UUID:     uuid,
		Address:  host,
		Port:     uint32(portInt),
		Flow:     q.Get("flow"),
		Security: q.Get("security"),
		SNI:      q.Get("sni"),
		FP:       q.Get("fp"),
		Type:     q.Get("type"),
		Path:     q.Get("path"),
		Host:     q.Get("host"),
		PBK:      q.Get("pbk"),
		SID:      q.Get("sid"),
		SpiderX:  q.Get("spider-x"),
	}, nil
}

func buildXrayConfig(localPort int, vless *VLESSConfig) (*core.Config, error) {
	// Inbound: SOCKS5
	socksSettings := json.RawMessage([]byte(`{"auth": "noauth", "udp": true}`))
	inbound := &conf.InboundDetourConfig{
		ListenOn: &conf.Address{Address: xnet.IPAddress(net.ParseIP("127.0.0.1"))},
		PortList: &conf.PortList{Range: []conf.PortRange{{From: uint32(localPort), To: uint32(localPort)}}},
		Protocol: "socks",
		Settings: &socksSettings,
		Tag:      "socks_in",
	}

	// Outbound: VLESS
	// Construct VLESS outbound settings JSON
	vlessSettingsJSON := fmt.Sprintf(`{
		"vnext": [{
			"address": "%s",
			"port": %d,
			"users": [{
				"id": "%s",
				"encryption": "none",
				"flow": "%s"
			}]
		}]
	}`, vless.Address, vless.Port, vless.UUID, vless.Flow)
	vlessSettings := json.RawMessage([]byte(vlessSettingsJSON))

	// Stream Settings
	streamSettings := make(map[string]interface{})
	streamSettings["network"] = vless.Type
	if vless.Type == "" {
		streamSettings["network"] = "tcp"
	}
	streamSettings["security"] = vless.Security
	
	// TLS/Reality Settings
	if vless.Security == "tls" || vless.Security == "reality" {
		tlsSettings := make(map[string]interface{})
		tlsSettings["serverName"] = vless.SNI
		if vless.FP != "" {
			tlsSettings["fingerprint"] = vless.FP
		}
		if vless.Security == "reality" {
			tlsSettings["publicKey"] = vless.PBK
			tlsSettings["shortId"] = vless.SID
			tlsSettings["show"] = false
			tlsSettings["spiderX"] = vless.SpiderX
		}
		streamSettings[vless.Security+"Settings"] = tlsSettings
	}

	// Transport Settings (ws, grpc, etc)
	if vless.Type == "ws" {
		wsSettings := make(map[string]interface{})
		if vless.Path != "" {
			wsSettings["path"] = vless.Path
		}
		if vless.Host != "" {
			wsSettings["headers"] = map[string]string{"Host": vless.Host}
		}
		streamSettings["wsSettings"] = wsSettings
	} else if vless.Type == "grpc" {
		grpcSettings := make(map[string]interface{})
		if vless.Path != "" {
			grpcSettings["serviceName"] = vless.Path
		}
		streamSettings["grpcSettings"] = grpcSettings
	}

	streamBytes, _ := json.Marshal(streamSettings)
	var streamConfig conf.StreamConfig
	if err := json.Unmarshal(streamBytes, &streamConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stream config: %v", err)
	}
	
	outbound := &conf.OutboundDetourConfig{
		Protocol: "vless",
		Settings: &vlessSettings,
		StreamSetting: &streamConfig,
		Tag: "proxy_out",
	}

	// Build full config using conf helper
	pbConfig, err := (&conf.Config{
		InboundConfigs: []conf.InboundDetourConfig{*inbound},
		OutboundConfigs: []conf.OutboundDetourConfig{*outbound},
	}).Build()

	return pbConfig, err
}

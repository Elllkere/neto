package singbox

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/elllkere/neto/internal/config"
)

func TestGenerateUsesModernFakeIPServer(t *testing.T) {
	cfg := config.Defaults()
	out, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatal(err)
	}
	raw := string(out)
	if !strings.Contains(raw, `"type": "fakeip"`) || !strings.Contains(raw, `"tag": "fakeip"`) {
		t.Fatalf("fakeip server missing:\n%s", raw)
	}
	if strings.Contains(raw, `"fake-ip"`) || strings.Contains(raw, `"fake_ip"`) {
		t.Fatalf("legacy fake-ip config detected:\n%s", raw)
	}
	if !strings.Contains(raw, `"listen_port": 15353`) || !strings.Contains(raw, `"listen_port": 16001`) {
		t.Fatalf("expected DNS and TProxy listeners:\n%s", raw)
	}
	if !strings.Contains(raw, `"default_domain_resolver": "local"`) {
		t.Fatalf("expected route.default_domain_resolver:\n%s", raw)
	}
	if strings.Contains(raw, `"override_destination"`) || strings.Contains(raw, `"sniff": true`) {
		t.Fatalf("generated config contains sing-box 1.13-incompatible sniff fields:\n%s", raw)
	}
	if !strings.Contains(raw, `"store_fakeip": true`) {
		t.Fatalf("expected fakeip cache persistence:\n%s", raw)
	}
}

func TestGenerateBuiltinOutbounds(t *testing.T) {
	cfg := config.Defaults()
	direct := generatedOutbound(t, cfg, "direct")
	blocked := generatedOutbound(t, cfg, "blocked")
	if direct["type"] != "direct" {
		t.Fatalf("unexpected direct outbound: %+v", direct)
	}
	if blocked["type"] != "block" {
		t.Fatalf("unexpected blocked outbound: %+v", blocked)
	}
}

func TestGenerateRoutesCapturedTrafficToReferencedProxyOutbound(t *testing.T) {
	cfg := config.Defaults()
	cfg.Outbounds = []config.Outbound{{
		Enabled: true,
		Tag:     "test",
		Type:    "vless",
		Server:  "example.com",
		Port:    443,
		UUID:    "a3482e88-686a-4a58-8126-99c9df64b060",
		TLS:     true,
	}}
	cfg.Rules = []config.Rule{{
		Name:           "test",
		Enabled:        true,
		Priority:       100,
		Action:         "proxy",
		Outbound:       "test",
		DNSMode:        "fakeip",
		DomainContains: []string{"check-host"},
	}}
	route := generatedRoute(t, cfg)
	if route["final"] != "test" {
		t.Fatalf("captured traffic should use referenced proxy outbound, got route %+v", route)
	}
	rules := route["rules"].([]any)
	sniff := rules[0].(map[string]any)
	if sniff["action"] != "sniff" {
		t.Fatalf("sniff action missing before proxy outbound, got %+v", sniff)
	}
	if _, ok := sniff["override_destination"]; ok {
		t.Fatalf("override_destination is not accepted by sing-box 1.13 route action, got %+v", sniff)
	}
}

func TestGenerateKeepsDirectFinalWhenProxyRuleUsesDirect(t *testing.T) {
	cfg := config.Defaults()
	cfg.Outbounds = []config.Outbound{{
		Enabled: true,
		Tag:     "test",
		Type:    "vless",
		Server:  "example.com",
		Port:    443,
		UUID:    "a3482e88-686a-4a58-8126-99c9df64b060",
		TLS:     true,
	}}
	cfg.Rules = []config.Rule{{
		Name:           "test",
		Enabled:        true,
		Priority:       100,
		Action:         "proxy",
		Outbound:       "direct",
		DNSMode:        "fakeip",
		DomainContains: []string{"check-host"},
	}}
	route := generatedRoute(t, cfg)
	if route["final"] != "direct" {
		t.Fatalf("direct-selected proxy rule should keep direct final, got route %+v", route)
	}
}

func TestGenerateGlobalModeUsesFirstCustomOutbound(t *testing.T) {
	cfg := config.Defaults()
	cfg.Main.RoutingMode = "global"
	cfg.Outbounds = []config.Outbound{{
		Enabled: true,
		Tag:     "test",
		Type:    "vless",
		Server:  "example.com",
		Port:    443,
		UUID:    "a3482e88-686a-4a58-8126-99c9df64b060",
		TLS:     true,
	}}
	route := generatedRoute(t, cfg)
	if route["final"] != "test" {
		t.Fatalf("global captured traffic should use first custom outbound, got route %+v", route)
	}
}

func TestGenerateVLESSOutbound(t *testing.T) {
	cfg := config.Defaults()
	cfg.Outbounds = []config.Outbound{{
		Enabled:    true,
		Tag:        "my_vless",
		Type:       "vless",
		Server:     "example.com",
		Port:       443,
		UUID:       "a3482e88-686a-4a58-8126-99c9df64b060",
		Flow:       "xtls-rprx-vision",
		TLS:        true,
		ServerName: "example.com",
		Transport:  "tcp",
	}}
	proxy := generatedOutbound(t, cfg, "my_vless")
	if proxy["type"] != "vless" || proxy["server"] != "example.com" || int(proxy["server_port"].(float64)) != 443 {
		t.Fatalf("unexpected vless outbound: %+v", proxy)
	}
	if proxy["uuid"] != "a3482e88-686a-4a58-8126-99c9df64b060" || proxy["flow"] != "xtls-rprx-vision" || proxy["network"] != "tcp" {
		t.Fatalf("missing vless fields: %+v", proxy)
	}
	tls := proxy["tls"].(map[string]any)
	if tls["enabled"] != true || tls["server_name"] != "example.com" {
		t.Fatalf("unexpected tls: %+v", tls)
	}
}

func TestGenerateVLESSRealityOutbound(t *testing.T) {
	cfg := config.Defaults()
	cfg.Outbounds = []config.Outbound{{
		Enabled:          true,
		Tag:              "my_vless",
		Type:             "vless",
		Server:           "example.com",
		Port:             443,
		UUID:             "a3482e88-686a-4a58-8126-99c9df64b060",
		Flow:             "xtls-rprx-vision",
		ServerName:       "www.example.com",
		Reality:          true,
		RealityPublicKey: "public-key",
		RealityShortID:   "0123456789abcdef",
		ALPN:             []string{"h2", "http/1.1"},
	}}
	proxy := generatedOutbound(t, cfg, "my_vless")
	tls := proxy["tls"].(map[string]any)
	reality := tls["reality"].(map[string]any)
	if tls["enabled"] != true || tls["server_name"] != "www.example.com" {
		t.Fatalf("unexpected tls: %+v", tls)
	}
	if reality["enabled"] != true || reality["public_key"] != "public-key" || reality["short_id"] != "0123456789abcdef" {
		t.Fatalf("unexpected reality: %+v", reality)
	}
	alpn := tls["alpn"].([]any)
	if len(alpn) != 2 || alpn[0] != "h2" || alpn[1] != "http/1.1" {
		t.Fatalf("unexpected alpn: %+v", alpn)
	}
}

func TestGenerateVLESSAdvancedTLSAndTransport(t *testing.T) {
	cfg := config.Defaults()
	cfg.Outbounds = []config.Outbound{{
		Enabled:           true,
		Tag:               "my_vless",
		Type:              "vless",
		Server:            "example.com",
		Port:              443,
		UUID:              "a3482e88-686a-4a58-8126-99c9df64b060",
		Flow:              "xtls-rprx-vision",
		TLS:               true,
		ServerName:        "www.example.com",
		ALPN:              []string{"h2", "http/1.1"},
		TLSMinVersion:     "1.2",
		TLSMaxVersion:     "1.3",
		TLSCipherSuites:   []string{"TLS_AES_128_GCM_SHA256"},
		ECH:               true,
		ECHConfig:         []string{"ech-config"},
		ECHConfigPath:     "/etc/neto/ech.pem",
		UTLSFingerprint:   "chrome",
		Reality:           true,
		RealityPublicKey:  "public-key",
		Transport:         "ws",
		WSHost:            "front.example.com",
		WSPath:            "/ws",
		WSEarlyData:       2048,
		WSEarlyDataHeader: "Sec-WebSocket-Protocol",
		PacketEncoding:    "xudp",
	}}
	proxy := generatedOutbound(t, cfg, "my_vless")
	if proxy["packet_encoding"] != "xudp" {
		t.Fatalf("missing packet encoding: %+v", proxy)
	}
	tls := proxy["tls"].(map[string]any)
	if tls["min_version"] != "1.2" || tls["max_version"] != "1.3" {
		t.Fatalf("missing TLS versions: %+v", tls)
	}
	if tls["utls"].(map[string]any)["fingerprint"] != "chrome" {
		t.Fatalf("missing uTLS: %+v", tls)
	}
	if tls["ech"].(map[string]any)["config_path"] != "/etc/neto/ech.pem" {
		t.Fatalf("missing ECH: %+v", tls)
	}
	ciphers := tls["cipher_suites"].([]any)
	if len(ciphers) != 1 || ciphers[0] != "TLS_AES_128_GCM_SHA256" {
		t.Fatalf("missing ciphers: %+v", tls)
	}
	transport := proxy["transport"].(map[string]any)
	if transport["type"] != "ws" || transport["path"] != "/ws" || int(transport["max_early_data"].(float64)) != 2048 {
		t.Fatalf("unexpected transport: %+v", transport)
	}
	headers := transport["headers"].(map[string]any)
	if headers["Host"] != "front.example.com" {
		t.Fatalf("missing websocket host header: %+v", transport)
	}
}

func TestGenerateHysteria2Outbound(t *testing.T) {
	cfg := config.Defaults()
	cfg.Outbounds = []config.Outbound{{
		Enabled:    true,
		Tag:        "my_hy2",
		Type:       "hysteria2",
		Server:     "example.com",
		Port:       443,
		Password:   "hy2-secret",
		ServerName: "example.com",
		Insecure:   true,
	}}
	proxy := generatedOutbound(t, cfg, "my_hy2")
	if proxy["type"] != "hysteria2" || proxy["password"] != "hy2-secret" {
		t.Fatalf("unexpected hysteria2 outbound: %+v", proxy)
	}
	tls := proxy["tls"].(map[string]any)
	if tls["enabled"] != true || tls["server_name"] != "example.com" || tls["insecure"] != true {
		t.Fatalf("unexpected hysteria2 tls: %+v", tls)
	}
}

func TestGenerateHysteria2AdvancedFields(t *testing.T) {
	cfg := config.Defaults()
	cfg.Outbounds = []config.Outbound{{
		Enabled:              true,
		Tag:                  "my_hy2",
		Type:                 "hysteria2",
		Server:               "example.com",
		Port:                 443,
		Password:             "hy2-secret",
		HysteriaObfsType:     "salamander",
		HysteriaObfsPassword: "obfs-secret",
		HysteriaUpMbps:       20,
		HysteriaDownMbps:     100,
	}}
	proxy := generatedOutbound(t, cfg, "my_hy2")
	obfs := proxy["obfs"].(map[string]any)
	if obfs["type"] != "salamander" || obfs["password"] != "obfs-secret" || int(proxy["up_mbps"].(float64)) != 20 || int(proxy["down_mbps"].(float64)) != 100 {
		t.Fatalf("unexpected hysteria2 advanced fields: %+v", proxy)
	}
}

func TestGenerateShadowsocksOutbound(t *testing.T) {
	cfg := config.Defaults()
	cfg.Outbounds = []config.Outbound{{
		Enabled:  true,
		Tag:      "my_ss",
		Type:     "shadowsocks",
		Server:   "example.com",
		Port:     8388,
		Method:   "2022-blake3-aes-128-gcm",
		Password: "ss-secret",
	}}
	proxy := generatedOutbound(t, cfg, "my_ss")
	if proxy["type"] != "shadowsocks" || proxy["method"] != "2022-blake3-aes-128-gcm" || proxy["password"] != "ss-secret" {
		t.Fatalf("unexpected shadowsocks outbound: %+v", proxy)
	}
}

func TestGenerateTrojanOutbound(t *testing.T) {
	cfg := config.Defaults()
	cfg.Outbounds = []config.Outbound{{
		Enabled:    true,
		Tag:        "my_trojan",
		Type:       "trojan",
		Server:     "example.com",
		Port:       443,
		Password:   "trojan-secret",
		TLS:        true,
		ServerName: "example.com",
	}}
	proxy := generatedOutbound(t, cfg, "my_trojan")
	if proxy["type"] != "trojan" || proxy["password"] != "trojan-secret" {
		t.Fatalf("unexpected trojan outbound: %+v", proxy)
	}
	tls := proxy["tls"].(map[string]any)
	if tls["enabled"] != true || tls["server_name"] != "example.com" {
		t.Fatalf("unexpected trojan tls: %+v", tls)
	}
}

func TestCompareVersion(t *testing.T) {
	if compareVersion("1.11.9", MinimumVersion) >= 0 {
		t.Fatal("1.11.9 should be unsupported")
	}
	if compareVersion("1.12.0", MinimumVersion) < 0 {
		t.Fatal("1.12.0 should be supported")
	}
	if compareVersion("1.13.12", MinimumVersion) < 0 {
		t.Fatal("1.13.12 should be supported")
	}
}

func generatedOutbound(t *testing.T, cfg config.Config, tag string) map[string]any {
	t.Helper()
	out, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Outbounds []map[string]any `json:"outbounds"`
	}
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, outbound := range decoded.Outbounds {
		if outbound["tag"] == tag {
			return outbound
		}
	}
	t.Fatalf("outbound %q not found in %s", tag, string(out))
	return nil
}

func generatedRoute(t *testing.T, cfg config.Config) map[string]any {
	t.Helper()
	out, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Route map[string]any `json:"route"`
	}
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded.Route
}

package singbox

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/elllkere/neto/internal/config"
)

const MinimumVersion = "1.12.0"

type Config struct {
	Log          map[string]any `json:"log,omitempty"`
	DNS          DNS            `json:"dns"`
	Inbounds     []any          `json:"inbounds"`
	Outbounds    []any          `json:"outbounds"`
	Route        Route          `json:"route"`
	Experimental map[string]any `json:"experimental,omitempty"`
}

type DNS struct {
	Servers          []any  `json:"servers"`
	Rules            []any  `json:"rules,omitempty"`
	Final            string `json:"final,omitempty"`
	Strategy         string `json:"strategy,omitempty"`
	IndependentCache bool   `json:"independent_cache,omitempty"`
}

type Route struct {
	Rules                 []any  `json:"rules,omitempty"`
	Final                 string `json:"final,omitempty"`
	DefaultDomainResolver string `json:"default_domain_resolver,omitempty"`
}

func Generate(cfg config.Config) ([]byte, error) {
	fakeDNSHost, fakeDNSPort, err := splitHostPort(cfg.Main.SingBoxDNSFakeIPAddr())
	if err != nil {
		return nil, err
	}
	realDirectDNSHost, realDirectDNSPort, err := splitHostPort(cfg.Main.SingBoxDNSRealDirectAddr())
	if err != nil {
		return nil, err
	}
	realProxyDNSHost, realProxyDNSPort, err := splitHostPort(cfg.Main.SingBoxDNSRealProxyAddr())
	if err != nil {
		return nil, err
	}

	upstream := cfg.Main.DNSUpstream()
	dnsServers := []any{
		encodeDNSServer("real-direct", upstream, ""),
		encodeDNSServer("real-proxy", upstream, DNSProxyOutbound(cfg)),
		map[string]any{
			"tag":         "fakeip",
			"type":        "fakeip",
			"inet4_range": cfg.Main.FakeIPRange,
		},
	}
	if needsBootstrap(upstream) {
		dnsServers = append(dnsServers, map[string]any{
			"tag":         "bootstrap",
			"type":        "udp",
			"server":      "1.1.1.1",
			"server_port": 53,
		})
	}

	doc := Config{
		Log: map[string]any{
			"level":     "warn",
			"timestamp": true,
		},
		DNS: DNS{
			Servers: dnsServers,
			Rules: []any{
				map[string]any{
					"inbound": []string{"dns-fakeip-in"},
					"action":  "route",
					"server":  "fakeip",
				},
				map[string]any{
					"inbound": []string{"dns-real-direct-in"},
					"action":  "route",
					"server":  "real-direct",
				},
				map[string]any{
					"inbound": []string{"dns-real-proxy-in"},
					"action":  "route",
					"server":  "real-proxy",
				},
			},
			Final:            "real-direct",
			Strategy:         "prefer_ipv4",
			IndependentCache: true,
		},
		Inbounds: []any{
			map[string]any{
				"type":        "direct",
				"tag":         "dns-fakeip-in",
				"listen":      fakeDNSHost,
				"listen_port": fakeDNSPort,
			},
			map[string]any{
				"type":        "direct",
				"tag":         "dns-real-direct-in",
				"listen":      realDirectDNSHost,
				"listen_port": realDirectDNSPort,
			},
			map[string]any{
				"type":        "direct",
				"tag":         "dns-real-proxy-in",
				"listen":      realProxyDNSHost,
				"listen_port": realProxyDNSPort,
			},
			map[string]any{
				"type":        "tproxy",
				"tag":         "tproxy-in",
				"listen":      "127.0.0.1",
				"listen_port": cfg.Main.TProxyPort,
			},
		},
		Outbounds: nil,
		Route: Route{
			Rules: []any{
				map[string]any{"action": "sniff"},
				map[string]any{"protocol": "dns", "action": "hijack-dns"},
			},
			Final:                 SelectedProxyOutbound(cfg),
			DefaultDomainResolver: "real-direct",
		},
	}
	if cfg.Main.FakeIPEnabled {
		doc.Experimental = map[string]any{
			"cache_file": map[string]any{
				"enabled":      true,
				"path":         "/tmp/neto/sing-box-cache.db",
				"store_fakeip": true,
			},
		}
	}

	outbounds, err := generateOutbounds(cfg)
	if err != nil {
		return nil, err
	}
	doc.Outbounds = outbounds

	return json.MarshalIndent(doc, "", "  ")
}

func GenerateProxyClient(cfg config.Config, outboundTag string, listenPort int) ([]byte, error) {
	if listenPort <= 0 || listenPort > 65535 {
		return nil, fmt.Errorf("invalid listen port %d", listenPort)
	}
	outbound, err := findClientOutbound(cfg, outboundTag)
	if err != nil {
		return nil, err
	}
	doc := map[string]any{
		"log": map[string]any{
			"level":     "warn",
			"timestamp": true,
		},
		"inbounds": []any{
			map[string]any{
				"type":        "mixed",
				"tag":         "mixed-in",
				"listen":      "127.0.0.1",
				"listen_port": listenPort,
			},
		},
		"outbounds": []any{
			map[string]any{"type": "direct", "tag": config.BuiltinDirectOutbound},
			encodeOutbound(outbound),
		},
		"route": map[string]any{
			"final": outbound.Tag,
		},
	}
	return json.MarshalIndent(doc, "", "  ")
}

func encodeDNSServer(tag string, upstream config.DNSUpstream, detour string) map[string]any {
	item := map[string]any{
		"tag":         tag,
		"type":        upstream.Protocol,
		"server":      upstream.Host,
		"server_port": upstream.Port,
	}
	if detour = dnsServerDetour(detour); detour != "" {
		item["detour"] = detour
	}
	if needsBootstrap(upstream) {
		item["domain_resolver"] = "bootstrap"
	}
	switch upstream.Protocol {
	case "tls":
		if upstream.TLSName != "" {
			item["tls"] = map[string]any{"server_name": upstream.TLSName}
		}
	case "https":
		if upstream.Path != "" {
			item["path"] = upstream.Path
		}
		if upstream.TLSName != "" {
			item["tls"] = map[string]any{"server_name": upstream.TLSName}
		}
	}
	return item
}

func dnsServerDetour(detour string) string {
	detour = strings.TrimSpace(detour)
	switch detour {
	case "", config.BuiltinDirectOutbound, config.BuiltinBlockedOutbound, "block", "proxy_default":
		return ""
	default:
		return detour
	}
}

func needsBootstrap(upstream config.DNSUpstream) bool {
	return upstream.Host != "" && net.ParseIP(strings.Trim(upstream.Host, "[]")) == nil
}

func findClientOutbound(cfg config.Config, outboundTag string) (config.Outbound, error) {
	outboundTag = strings.TrimSpace(outboundTag)
	var first config.Outbound
	for i, outbound := range cfg.EnabledCustomOutbounds() {
		if i == 0 {
			first = outbound
		}
		if outboundTag != "" && outbound.Tag == outboundTag {
			return outbound, nil
		}
	}
	if outboundTag != "" {
		return config.Outbound{}, fmt.Errorf("update_outbound %q not found", outboundTag)
	}
	if first.Tag == "" {
		return config.Outbound{}, fmt.Errorf("update_via proxy requires update_outbound or at least one custom outbound")
	}
	return first, nil
}

func generateOutbounds(cfg config.Config) ([]any, error) {
	used := map[string]struct{}{}
	var out []any
	add := func(item map[string]any) {
		tag, _ := item["tag"].(string)
		if tag == "" {
			return
		}
		if _, ok := used[tag]; ok {
			return
		}
		used[tag] = struct{}{}
		out = append(out, item)
	}

	add(map[string]any{"type": "direct", "tag": config.BuiltinDirectOutbound})
	add(map[string]any{"type": "block", "tag": config.BuiltinBlockedOutbound})
	for _, outbound := range cfg.EnabledCustomOutbounds() {
		add(encodeOutbound(outbound))
	}
	return out, nil
}

func SelectedProxyOutbound(cfg config.Config) string {
	allowed := cfg.AllowedOutboundTags()
	for _, rule := range cfg.Rules {
		if !rule.Enabled || rule.Action != "proxy" {
			continue
		}
		tag := strings.TrimSpace(rule.Outbound)
		if tag == "" || tag == config.BuiltinDirectOutbound || tag == config.BuiltinBlockedOutbound {
			continue
		}
		if _, ok := allowed[tag]; ok {
			return tag
		}
	}
	if cfg.Main.RoutingMode == "global" || hasProxyClient(cfg) {
		for _, outbound := range cfg.EnabledCustomOutbounds() {
			return outbound.Tag
		}
	}
	return config.BuiltinDirectOutbound
}

func DNSProxyOutbound(cfg config.Config) string {
	tag := strings.TrimSpace(cfg.Main.RealDNSOutbound)
	if tag != "" && tag != config.BuiltinDirectOutbound && tag != config.BuiltinBlockedOutbound && tag != "block" && tag != "proxy_default" {
		for _, outbound := range cfg.EnabledCustomOutbounds() {
			if outbound.Tag == tag {
				return tag
			}
		}
	}
	return SelectedProxyOutbound(cfg)
}

func hasProxyClient(cfg config.Config) bool {
	for _, client := range cfg.Clients {
		if client.Policy == "proxy" {
			return true
		}
	}
	return false
}

func encodeOutbound(outbound config.Outbound) map[string]any {
	switch outbound.Type {
	case "vless":
		doc := map[string]any{
			"type":        "vless",
			"tag":         outbound.Tag,
			"server":      outbound.Server,
			"server_port": outbound.Port,
			"uuid":        outbound.UUID,
		}
		if outbound.Flow != "" {
			doc["flow"] = outbound.Flow
		}
		if outbound.Transport != "" {
			if outbound.Transport == "tcp" {
				doc["network"] = outbound.Transport
			} else if transport := v2rayTransportConfig(outbound); transport != nil {
				doc["transport"] = transport
			}
		}
		if outbound.PacketEncoding != "" {
			doc["packet_encoding"] = outbound.PacketEncoding
		}
		if tls := tlsConfig(outbound, outbound.TLS || outbound.Reality); tls != nil {
			doc["tls"] = tls
		}
		return doc
	case "hysteria2":
		doc := map[string]any{
			"type":        "hysteria2",
			"tag":         outbound.Tag,
			"server":      outbound.Server,
			"server_port": outbound.Port,
			"password":    outbound.Password,
			"tls":         tlsConfig(outbound, true),
		}
		if outbound.HysteriaObfsType != "" {
			doc["obfs"] = map[string]any{
				"type":     outbound.HysteriaObfsType,
				"password": outbound.HysteriaObfsPassword,
			}
		}
		if outbound.HysteriaUpMbps > 0 {
			doc["up_mbps"] = outbound.HysteriaUpMbps
		}
		if outbound.HysteriaDownMbps > 0 {
			doc["down_mbps"] = outbound.HysteriaDownMbps
		}
		return doc
	case "shadowsocks":
		return map[string]any{
			"type":        "shadowsocks",
			"tag":         outbound.Tag,
			"server":      outbound.Server,
			"server_port": outbound.Port,
			"method":      outbound.Method,
			"password":    outbound.Password,
		}
	case "trojan":
		doc := map[string]any{
			"type":        "trojan",
			"tag":         outbound.Tag,
			"server":      outbound.Server,
			"server_port": outbound.Port,
			"password":    outbound.Password,
		}
		if tls := tlsConfig(outbound, outbound.TLS); tls != nil {
			doc["tls"] = tls
		}
		if transport := v2rayTransportConfig(outbound); transport != nil {
			doc["transport"] = transport
		}
		return doc
	default:
		return map[string]any{
			"type": "direct",
			"tag":  outbound.Tag,
		}
	}
}

func tlsConfig(outbound config.Outbound, enabled bool) map[string]any {
	if !enabled {
		return nil
	}
	tls := map[string]any{"enabled": true}
	if outbound.ServerName != "" {
		tls["server_name"] = outbound.ServerName
	}
	if outbound.Insecure {
		tls["insecure"] = true
	}
	if len(outbound.ALPN) > 0 {
		tls["alpn"] = outbound.ALPN
	}
	if outbound.TLSMinVersion != "" {
		tls["min_version"] = outbound.TLSMinVersion
	}
	if outbound.TLSMaxVersion != "" {
		tls["max_version"] = outbound.TLSMaxVersion
	}
	if len(outbound.TLSCipherSuites) > 0 {
		tls["cipher_suites"] = outbound.TLSCipherSuites
	}
	if outbound.ECH {
		ech := map[string]any{"enabled": true}
		if len(outbound.ECHConfig) > 0 {
			ech["config"] = outbound.ECHConfig
		}
		if outbound.ECHConfigPath != "" {
			ech["config_path"] = outbound.ECHConfigPath
		}
		tls["ech"] = ech
	}
	if outbound.UTLSFingerprint != "" {
		tls["utls"] = map[string]any{
			"enabled":     true,
			"fingerprint": outbound.UTLSFingerprint,
		}
	}
	if outbound.Reality {
		reality := map[string]any{
			"enabled":    true,
			"public_key": outbound.RealityPublicKey,
		}
		if outbound.RealityShortID != "" {
			reality["short_id"] = outbound.RealityShortID
		}
		tls["reality"] = reality
	}
	return tls
}

func v2rayTransportConfig(outbound config.Outbound) map[string]any {
	if outbound.Transport == "" || outbound.Transport == "tcp" {
		return nil
	}
	transport := map[string]any{"type": outbound.Transport}
	switch outbound.Transport {
	case "grpc":
		if outbound.GRPCServiceName != "" {
			transport["service_name"] = outbound.GRPCServiceName
		}
	case "http":
		if len(outbound.HTTPHost) > 0 {
			transport["host"] = outbound.HTTPHost
		}
		if outbound.HTTPPath != "" {
			transport["path"] = outbound.HTTPPath
		}
		if outbound.HTTPMethod != "" {
			transport["method"] = outbound.HTTPMethod
		}
	case "httpupgrade":
		if outbound.HTTPUpgradeHost != "" {
			transport["host"] = outbound.HTTPUpgradeHost
		}
		if outbound.HTTPPath != "" {
			transport["path"] = outbound.HTTPPath
		}
	case "ws":
		if outbound.WSPath != "" {
			transport["path"] = outbound.WSPath
		}
		if outbound.WSHost != "" {
			transport["headers"] = map[string]any{"Host": outbound.WSHost}
		}
		if outbound.WSEarlyData > 0 {
			transport["max_early_data"] = outbound.WSEarlyData
		}
		if outbound.WSEarlyDataHeader != "" {
			transport["early_data_header_name"] = outbound.WSEarlyDataHeader
		}
	case "quic":
	default:
		return nil
	}
	return transport
}

func CheckBinary(bin string, configPath string) error {
	versionOut, err := exec.Command(bin, "version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s version failed: %w: %s", bin, err, strings.TrimSpace(string(versionOut)))
	}
	version, err := extractVersion(string(versionOut))
	if err != nil {
		return err
	}
	if compareVersion(version, MinimumVersion) < 0 {
		return fmt.Errorf("%s version %s is unsupported, need >= %s", bin, version, MinimumVersion)
	}

	checkOut, err := exec.Command(bin, "check", "-c", configPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s check -c %s failed: %w: %s", bin, configPath, err, strings.TrimSpace(string(checkOut)))
	}
	return nil
}

func BinaryExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir() && st.Mode()&0111 != 0
}

func splitHostPort(addr string) (string, uint16, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid singbox_dns %q: %w", addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("invalid singbox_dns port %q", portStr)
	}
	return host, uint16(port), nil
}

var versionRE = regexp.MustCompile(`\b([0-9]+)\.([0-9]+)\.([0-9]+)\b`)

func extractVersion(s string) (string, error) {
	m := versionRE.FindStringSubmatch(s)
	if m == nil {
		return "", fmt.Errorf("could not parse sing-box version from %q", strings.TrimSpace(s))
	}
	return m[0], nil
}

func compareVersion(a, b string) int {
	pa := parseVersion(a)
	pb := parseVersion(b)
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

func parseVersion(v string) [3]int {
	var out [3]int
	parts := strings.Split(v, ".")
	for i := 0; i < len(parts) && i < 3; i++ {
		out[i], _ = strconv.Atoi(parts[i])
	}
	return out
}

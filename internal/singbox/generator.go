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
	Log       map[string]any `json:"log,omitempty"`
	DNS       DNS            `json:"dns"`
	Inbounds  []any          `json:"inbounds"`
	Outbounds []any          `json:"outbounds"`
	Route     Route          `json:"route"`
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
	dnsHost, dnsPort, err := splitHostPort(cfg.Main.SingBoxDNS)
	if err != nil {
		return nil, err
	}
	realDNSHost, realDNSPort, err := splitHostPort(cfg.Main.RealDNSUpstream)
	if err != nil {
		return nil, err
	}

	doc := Config{
		Log: map[string]any{
			"level":     "info",
			"timestamp": true,
		},
		DNS: DNS{
			Servers: []any{
				map[string]any{
					"tag":         "local",
					"type":        "udp",
					"server":      realDNSHost,
					"server_port": realDNSPort,
				},
				map[string]any{
					"tag":         "fakeip",
					"type":        "fakeip",
					"inet4_range": cfg.Main.FakeIPRange,
				},
			},
			Rules: []any{
				map[string]any{
					"query_type": []string{"A", "AAAA"},
					"server":     "fakeip",
				},
			},
			Final:            "local",
			Strategy:         "prefer_ipv4",
			IndependentCache: true,
		},
		Inbounds: []any{
			map[string]any{
				"type":        "direct",
				"tag":         "dns-in",
				"listen":      dnsHost,
				"listen_port": dnsPort,
			},
			map[string]any{
				"type":        "tproxy",
				"tag":         "tproxy-in",
				"listen":      "127.0.0.1",
				"listen_port": cfg.Main.TProxyPort,
				"sniff":       true,
			},
		},
		Outbounds: []any{
			map[string]any{
				"type": "direct",
				"tag":  "proxy_default",
			},
			map[string]any{
				"type": "direct",
				"tag":  "direct",
			},
			map[string]any{
				"type": "block",
				"tag":  "block",
			},
		},
		Route: Route{
			Rules: []any{
				map[string]any{"action": "sniff"},
				map[string]any{"protocol": "dns", "action": "hijack-dns"},
			},
			Final:                 "proxy_default",
			DefaultDomainResolver: "local",
		},
	}

	return json.MarshalIndent(doc, "", "  ")
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

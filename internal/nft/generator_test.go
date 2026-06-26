package nft

import (
	"strings"
	"testing"

	"github.com/elllkere/neto/internal/config"
	"github.com/elllkere/neto/internal/policy"
)

func TestGenerateOrder(t *testing.T) {
	cfg := config.Defaults()
	cfg.Main.NFTCounters = false
	cfg.Clients = []config.Client{
		{Name: "direct", IP: "192.168.8.50", Policy: "direct"},
		{Name: "proxy", IP: "192.168.8.100", Policy: "proxy"},
	}
	out, err := Generate(Input{
		Config:        cfg,
		ProviderCIDRs: policy.MustIPv4CIDRs("1.1.1.0/24"),
	})
	if err != nil {
		t.Fatal(err)
	}

	guard := strings.Index(out, "ip saddr @lan_subnets4 jump from_lan")
	fromLAN := strings.Index(out, "\tchain from_lan")
	direct := strings.Index(out, "ip saddr @direct_clients4 return")
	reserved := strings.Index(out, "ip daddr @reserved4 return")
	forced := strings.Index(out, "ip saddr @proxy_clients4 meta l4proto { tcp, udp } jump to_proxy_default")
	fakeip := strings.Index(out, "ip daddr 198.18.0.0/15 meta l4proto { tcp, udp } jump to_proxy_default")
	subnet := strings.Index(out, "ip daddr @proxy_default4 meta l4proto { tcp, udp } jump to_proxy_default")
	def := strings.Index(out, "\t\treturn\n\t}\n\tchain to_proxy_default")

	if !(guard >= 0 && guard < fromLAN && fromLAN < direct && direct < reserved && reserved < forced && forced < fakeip && fakeip < subnet && subnet < def) {
		t.Fatalf("unexpected rule order:\n%s", out)
	}
	if strings.Count(out, "1.1.1.0/24") != 1 {
		t.Fatalf("provider CIDR was not emitted once:\n%s", out)
	}
}

func TestGenerateUsesSetForManyProviderCIDRs(t *testing.T) {
	cfg := config.Defaults()
	cfg.Main.NFTCounters = false
	cidrs := make([]string, 0, 7000)
	for i := 0; i < 7000; i++ {
		cidrs = append(cidrs, "100."+string(rune('0'+(i%10)))+".0.0/24")
	}
	// Use deterministic non-overlapping input after the cheap size check above.
	var provider []string
	for a := 1; a <= 28; a++ {
		for b := 0; b < 250; b++ {
			provider = append(provider, "11."+itoa(a)+"."+itoa(b)+".0/24")
		}
	}
	out, err := Generate(Input{Config: cfg, ProviderCIDRs: policy.MustIPv4CIDRs(provider...)})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(out, "ip daddr @proxy_default4 meta l4proto { tcp, udp } jump to_proxy_default") != 1 {
		t.Fatalf("expected one set-based proxy rule:\n%s", out)
	}
	if strings.Count(out, "set proxy_default4") != 1 {
		t.Fatalf("expected one proxy_default4 set:\n%s", out)
	}
	if strings.Count(out, "jump to_proxy_default") > 3 {
		t.Fatalf("expected no per-CIDR proxy jump rules:\n%s", out)
	}
}

func TestGenerateCounters(t *testing.T) {
	cfg := config.Defaults()
	cfg.Main.NFTCounters = true
	out, err := Generate(Input{Config: cfg, ProviderCIDRs: policy.MustIPv4CIDRs("1.1.1.0/24")})
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range []string{
		"ip saddr @direct_clients4 counter return",
		"ip daddr @reserved4 counter return",
		"ip saddr @proxy_clients4 meta l4proto { tcp, udp } counter jump to_proxy_default",
		"ip daddr 198.18.0.0/15 meta l4proto { tcp, udp } counter jump to_proxy_default",
		"ip daddr @proxy_default4 meta l4proto { tcp, udp } counter jump to_proxy_default",
		"meta l4proto { tcp, udp } counter meta mark set",
	} {
		if !strings.Contains(out, rule) {
			t.Fatalf("missing counter rule %q:\n%s", rule, out)
		}
	}
}

func TestGenerateCustomModeNoGlobalCatchAll(t *testing.T) {
	cfg := config.Defaults()
	cfg.Main.NFTCounters = false
	cfg.Main.RoutingMode = "custom"
	out, err := Generate(Input{Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "\t\tmeta l4proto { tcp, udp } jump to_proxy_default\n") {
		t.Fatalf("custom mode must not include global catch-all:\n%s", out)
	}
	if !strings.Contains(out, "ip daddr 198.18.0.0/15 meta l4proto { tcp, udp } jump to_proxy_default") {
		t.Fatalf("custom mode must include fakeip proxy rule:\n%s", out)
	}
}

func TestGenerateGlobalModeCatchAll(t *testing.T) {
	cfg := config.Defaults()
	cfg.Main.NFTCounters = false
	cfg.Main.RoutingMode = "global"
	cfg.Clients = []config.Client{
		{Name: "direct", IP: "192.168.8.50", Policy: "direct"},
		{Name: "default", IP: "192.168.8.60", Policy: "default"},
	}
	out, err := Generate(Input{Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	guard := strings.Index(out, "ip saddr @lan_subnets4 jump from_lan")
	nonLANReturn := strings.Index(out, "ip saddr @lan_subnets4 jump from_lan\n\t\treturn\n\t}\n\tchain from_lan")
	direct := strings.Index(out, "ip saddr @direct_clients4 return")
	reserved := strings.Index(out, "ip daddr @reserved4 return")
	global := strings.Index(out, "\t\tmeta l4proto { tcp, udp } jump to_proxy_default\n")
	if !(guard >= 0 && nonLANReturn >= 0 && guard < direct && direct < reserved && reserved < global) {
		t.Fatalf("global rule order is wrong:\n%s", out)
	}
	if strings.Contains(out, "ip daddr 198.18.0.0/15 meta l4proto") {
		t.Fatalf("global mode should not need fakeip destination rule:\n%s", out)
	}
}

func TestCustomClientDirectReturnsBeforeProxy(t *testing.T) {
	cfg := config.Defaults()
	cfg.Main.NFTCounters = false
	cfg.Main.RoutingMode = "custom"
	cfg.Clients = []config.Client{{Name: "pc", IP: "192.168.8.50", Policy: "direct"}}
	out, err := Generate(Input{Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	directRule := strings.Index(out, "ip saddr @direct_clients4 return")
	fakeRule := strings.Index(out, "ip daddr 198.18.0.0/15 meta l4proto { tcp, udp } jump to_proxy_default")
	if !(directRule >= 0 && fakeRule >= 0 && directRule < fakeRule) {
		t.Fatalf("direct client must return before fakeip proxy:\n%s", out)
	}
}

func TestCustomClientProxySourceRule(t *testing.T) {
	cfg := config.Defaults()
	cfg.Main.NFTCounters = false
	cfg.Main.RoutingMode = "custom"
	cfg.Clients = []config.Client{{Name: "pc", IP: "192.168.8.100", Policy: "proxy"}}
	out, err := Generate(Input{Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "set proxy_clients4") || !strings.Contains(out, "192.168.8.100") {
		t.Fatalf("proxy client set missing:\n%s", out)
	}
	if !strings.Contains(out, "ip saddr @proxy_clients4 meta l4proto { tcp, udp } jump to_proxy_default") {
		t.Fatalf("proxy client source rule missing:\n%s", out)
	}
}

func TestGlobalClientDefaultUsesCatchAll(t *testing.T) {
	cfg := config.Defaults()
	cfg.Main.NFTCounters = false
	cfg.Main.RoutingMode = "global"
	cfg.Clients = []config.Client{{Name: "pc", IP: "192.168.8.60", Policy: "default"}}
	out, err := Generate(Input{Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "192.168.8.60") {
		t.Fatalf("default client must not be placed in direct/proxy sets:\n%s", out)
	}
	if !strings.Contains(out, "ip saddr @lan_subnets4 jump from_lan") {
		t.Fatalf("LAN guard missing:\n%s", out)
	}
	if !strings.Contains(out, "\t\tmeta l4proto { tcp, udp } jump to_proxy_default\n") {
		t.Fatalf("global catch-all missing:\n%s", out)
	}
}

func TestGenerateLANSubnetsSet(t *testing.T) {
	cfg := config.Defaults()
	cfg.Main.NFTCounters = false
	cfg.Main.LANSubnets = []string{"192.168.10.0/24", "192.168.8.0/24", "192.168.8.0/24"}
	out, err := Generate(Input{Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "set lan_subnets4") {
		t.Fatalf("lan_subnets4 set missing:\n%s", out)
	}
	if strings.Count(out, "192.168.8.0/24") != 1 || strings.Count(out, "192.168.10.0/24") != 1 {
		t.Fatalf("LAN subnets must be normalized and deduplicated:\n%s", out)
	}
	if !strings.Contains(out, "ip saddr @lan_subnets4 jump from_lan\n\t\treturn\n\t}\n\tchain from_lan") {
		t.Fatalf("non-LAN traffic must return before policy rules:\n%s", out)
	}
}

func TestGenerateLANIfaceGuard(t *testing.T) {
	cfg := config.Defaults()
	cfg.Main.NFTCounters = false
	cfg.Main.LANIfaces = []string{"br-lan", "lan0"}
	out, err := Generate(Input{Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `iifname { "br-lan", "lan0" } ip saddr @lan_subnets4 jump from_lan`) {
		t.Fatalf("LAN iface guard missing:\n%s", out)
	}
}

func TestGenerateGlobalNonLANReturnsBeforeCatchAll(t *testing.T) {
	cfg := config.Defaults()
	cfg.Main.NFTCounters = false
	cfg.Main.RoutingMode = "global"
	out, err := Generate(Input{Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	nonLANReturn := strings.Index(out, "\t\treturn\n\t}\n\tchain from_lan")
	global := strings.Index(out, "\t\tmeta l4proto { tcp, udp } jump to_proxy_default\n")
	if !(nonLANReturn >= 0 && global >= 0 && nonLANReturn < global) {
		t.Fatalf("non-LAN return must be before global proxy rule:\n%s", out)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

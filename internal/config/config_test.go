package config

import (
	"strings"
	"testing"
)

func TestParseInitialConfig(t *testing.T) {
	cfg, err := Parse(`
config main 'main'
	option enabled '1'
	option singbox_bin '/usr/libexec/neto/sing-box'
	option singbox_dns '127.0.0.1:15353'
	option tproxy_port '16001'
	option mark '0x101'
	option table '101'
	option fakeip_enabled '1'
	option fakeip_range '198.18.0.0/15'
	option default_action 'direct'

config client
	option name 'gaming_pc'
	option ip '192.168.8.50'
	option policy 'bypass'

config rule
	option name 'youtube'
	option enabled '1'
	option priority '100'
	option action 'proxy'
	option dns_mode 'fakeip'
	option outbound 'proxy_default'
	list domain_equals 'youtube.com'
	list domain_ends_with '.youtube.com'
	list domain_equals 'googlevideo.com'
	list domain_ends_with '.googlevideo.com'

config rule
	option name 'cloudflare'
	option enabled '1'
	option priority '200'
	option action 'proxy'
	option dns_mode 'real_ip'
	option outbound 'proxy_default'
	list file '/etc/neto/providers/cloudflare-v4.txt'
`)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Main.Enabled || cfg.Main.TProxyPort != 16001 || cfg.Main.Table != 101 {
		t.Fatalf("unexpected main config: %+v", cfg.Main)
	}
	if cfg.Main.RoutingMode != "custom" || cfg.Main.DefaultOutbound != "direct" {
		t.Fatalf("unexpected routing defaults: %+v", cfg.Main)
	}
	if len(cfg.Main.LANSubnets) != 1 || cfg.Main.LANSubnets[0] != "192.168.8.0/24" {
		t.Fatalf("unexpected LAN defaults: %+v", cfg.Main.LANSubnets)
	}
	if len(cfg.Clients) != 1 || cfg.Clients[0].Policy != "direct" {
		t.Fatalf("unexpected clients: %+v", cfg.Clients)
	}
	if len(cfg.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %+v", cfg.Rules)
	}
	if cfg.Rules[0].Action != "proxy" || cfg.Rules[0].DNSMode != "fakeip" || len(cfg.Rules[0].DomainEndsWith) != 2 || len(cfg.Rules[0].DomainEquals) != 2 {
		t.Fatalf("unexpected domain rule: %+v", cfg.Rules[0])
	}
	if cfg.Rules[1].Action != "proxy" || cfg.Rules[1].DNSMode != "real_ip" || len(cfg.Rules[1].Files) != 1 {
		t.Fatalf("unexpected provider rule: %+v", cfg.Rules[1])
	}
}

func TestClientPolicyAliases(t *testing.T) {
	cfg, err := Parse(`
config client
	option ip '192.168.8.10'
	option policy 'bypass'

config client
	option ip '192.168.8.11'
	option policy 'proxy_default'

config client
	option ip '192.168.8.12'
	option policy 'direct'

config client
	option ip '192.168.8.13'
	option policy 'default'
`)
	if err != nil {
		t.Fatal(err)
	}
	got := []string{cfg.Clients[0].Policy, cfg.Clients[1].Policy, cfg.Clients[2].Policy, cfg.Clients[3].Policy}
	want := []string{"direct", "proxy", "direct", "default"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if len(cfg.Warnings) != 2 {
		t.Fatalf("got warnings %v, want 2 alias warnings", cfg.Warnings)
	}
}

func TestRoutingModeOptions(t *testing.T) {
	cfg, err := Parse(`
config main 'main'
	option routing_mode 'global'
	option default_outbound 'direct'
	list lan_subnet '192.168.8.0/24'
	list lan_subnet '192.168.9.0/24'
	list lan_iface 'br-lan'
`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Main.RoutingMode != "global" || cfg.Main.DefaultOutbound != "direct" {
		t.Fatalf("unexpected main: %+v", cfg.Main)
	}
	if len(cfg.Main.LANSubnets) != 2 || cfg.Main.LANSubnets[1] != "192.168.9.0/24" {
		t.Fatalf("unexpected LAN subnets: %+v", cfg.Main.LANSubnets)
	}
	if len(cfg.Main.LANIfaces) != 1 || cfg.Main.LANIfaces[0] != "br-lan" {
		t.Fatalf("unexpected LAN ifaces: %+v", cfg.Main.LANIfaces)
	}
}

func TestParseOrderedRules(t *testing.T) {
	cfg, err := Parse(`
config rule
	option name 'proxy_youtube'
	option enabled '1'
	option priority '200'
	option action 'proxy'
	option outbound 'proxy_default'
	option dns_mode 'fakeip'
	list domain_contains 'YouTube'
	list exclude_domain_ends_with '.youtube.kz'

config rule
	option name 'direct_kz'
	option priority '100'
	option action 'direct'
	option dns_mode 'real_ip'
	list domain_equals 'youtube.kz'
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Rules) != 2 {
		t.Fatalf("unexpected rules: %+v", cfg.Rules)
	}
	if cfg.Rules[0].Name != "direct_kz" || cfg.Rules[1].Name != "proxy_youtube" {
		t.Fatalf("rules were not sorted by priority: %+v", cfg.Rules)
	}
	if cfg.Rules[1].DomainContains[0] != "youtube" || cfg.Rules[1].ExcludeDomainEndsWith[0] != ".youtube.kz" {
		t.Fatalf("domain lists were not normalized: %+v", cfg.Rules[1])
	}
}

func TestParseRuleAliases(t *testing.T) {
	cfg, err := Parse(`
config rule
	option name 'aliases'
	option action 'proxy'
	option dns_mode 'fakeip'
	list domain_exact 'Example.COM.'
	list domain_keyword 'tube'
	list domain_prefix 'www'
	list domain_suffix '.example.com.'
	list exclude_domain_exact 'bad.example.com'
	list exclude_domain_keyword 'ads'
	list exclude_domain_prefix 'old'
	list exclude_domain_suffix '.bad.example.com'
`)
	if err != nil {
		t.Fatal(err)
	}
	r := cfg.Rules[0]
	if r.DomainEquals[0] != "example.com" {
		t.Fatalf("domain_exact alias failed: %+v", r)
	}
	if r.DomainContains[0] != "tube" {
		t.Fatalf("domain_keyword alias failed: %+v", r)
	}
	if r.DomainStartsWith[0] != "www" {
		t.Fatalf("domain_prefix alias failed: %+v", r)
	}
	if r.DomainEndsWith[0] != ".example.com" {
		t.Fatalf("domain_suffix alias failed: %+v", r)
	}
	if r.ExcludeDomainEquals[0] != "bad.example.com" ||
		r.ExcludeDomainContains[0] != "ads" ||
		r.ExcludeDomainStartsWith[0] != "old" ||
		r.ExcludeDomainEndsWith[0] != ".bad.example.com" {
		t.Fatalf("exclude aliases failed: %+v", r)
	}
}

func TestParseIgnoresDeprecatedMatchAll(t *testing.T) {
	cfg, err := Parse(`
config rule
	option name 'old_match_all'
	option enabled '1'
	option action 'proxy'
	option dns_mode 'fakeip'
	option match_all '1'
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Rules) != 1 {
		t.Fatalf("unexpected rules: %+v", cfg.Rules)
	}
	if len(cfg.Warnings) != 1 || !strings.Contains(cfg.Warnings[0], "match_all is deprecated and ignored") {
		t.Fatalf("expected match_all warning, got %+v", cfg.Warnings)
	}
	if len(cfg.Rules[0].DomainEquals)+len(cfg.Rules[0].DomainContains)+len(cfg.Rules[0].DomainStartsWith)+len(cfg.Rules[0].DomainEndsWith)+len(cfg.Rules[0].Files) != 0 {
		t.Fatalf("match_all should not create include conditions: %+v", cfg.Rules[0])
	}
}

func TestParseRejectsInvalidLANSubnet(t *testing.T) {
	if _, err := Parse(`
config main 'main'
	list lan_subnet 'not-a-cidr'
`); err == nil {
		t.Fatal("expected invalid lan_subnet to be rejected")
	}
}

func TestParseRejectsDNSLoops(t *testing.T) {
	if _, err := Parse(`
config main 'main'
	option dns_listen '127.0.0.1:5353'
	option real_dns_upstream '127.0.0.1:5353'
`); err == nil {
		t.Fatal("expected real_dns_upstream loop to be rejected")
	}

	if _, err := Parse(`
config main 'main'
	option dns_listen '127.0.0.1:5353'
	option singbox_dns '127.0.0.1:5353'
`); err == nil {
		t.Fatal("expected singbox_dns loop to be rejected")
	}
}

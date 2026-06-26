package config

import "testing"

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

config domain_rule
	option name 'youtube'
	option mode 'fakeip'
	option outbound 'proxy_default'
	list suffix 'youtube.com'
	list suffix 'googlevideo.com'

config subnet_rule
	option name 'cloudflare'
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
	if len(cfg.DomainRules) != 1 || len(cfg.DomainRules[0].Suffixes) != 2 {
		t.Fatalf("unexpected domain rules: %+v", cfg.DomainRules)
	}
	if len(cfg.SubnetRules) != 1 || len(cfg.SubnetRules[0].Files) != 1 {
		t.Fatalf("unexpected subnet rules: %+v", cfg.SubnetRules)
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

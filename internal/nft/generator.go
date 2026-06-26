package nft

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/elllkere/neto/internal/config"
	"github.com/elllkere/neto/internal/policy"
)

type Input struct {
	Config        config.Config
	ProviderCIDRs []*net.IPNet
}

func Generate(in Input) (string, error) {
	cfg := in.Config
	directClients := collectClients(cfg, "direct")
	proxyClients := collectClients(cfg, "proxy")
	providerCIDRs := policy.NormalizeIPv4CIDRs(in.ProviderCIDRs)
	lanSubnets := policy.NormalizeIPv4CIDRs(policy.MustIPv4CIDRs(cfg.Main.LANSubnets...))
	reserved4 := reservedCIDRs(cfg)

	var b strings.Builder
	b.WriteString("table inet neto {\n")
	writeSet(&b, "lan_subnets4", policy.CIDRStrings(lanSubnets))
	writeSet(&b, "reserved4", policy.CIDRStrings(reserved4))
	writeSet(&b, "direct_clients4", directClients)
	writeSet(&b, "proxy_clients4", proxyClients)
	writeSet(&b, "proxy_default4", policy.CIDRStrings(providerCIDRs))
	b.WriteString("\tchain prerouting {\n")
	b.WriteString("\t\ttype filter hook prerouting priority mangle; policy accept;\n")
	b.WriteString(lanGuardRule(cfg))
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n")
	b.WriteString("\tchain from_lan {\n")
	b.WriteString(fmt.Sprintf("\t\tip saddr @direct_clients4%s return\n", counter(cfg)))
	b.WriteString(fmt.Sprintf("\t\tip daddr @reserved4%s return\n", counter(cfg)))
	b.WriteString(fmt.Sprintf("\t\tip saddr @proxy_clients4 meta l4proto { tcp, udp }%s jump to_proxy_default\n", counter(cfg)))
	switch cfg.Main.RoutingMode {
	case "global":
		b.WriteString(fmt.Sprintf("\t\tmeta l4proto { tcp, udp }%s jump to_proxy_default\n", counter(cfg)))
	default:
		if cfg.Main.FakeIPEnabled {
			b.WriteString(fmt.Sprintf("\t\tip daddr %s meta l4proto { tcp, udp }%s jump to_proxy_default\n", cfg.Main.FakeIPRange, counter(cfg)))
		}
		b.WriteString(fmt.Sprintf("\t\tip daddr @proxy_default4 meta l4proto { tcp, udp }%s jump to_proxy_default\n", counter(cfg)))
	}
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n")
	b.WriteString("\tchain to_proxy_default {\n")
	b.WriteString(fmt.Sprintf("\t\tmeta l4proto { tcp, udp }%s meta mark set %s tproxy ip to 127.0.0.1:%d accept\n", counter(cfg), cfg.Main.Mark, cfg.Main.TProxyPort))
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	return b.String(), nil
}

func lanGuardRule(cfg config.Config) string {
	if len(cfg.Main.LANIfaces) == 0 {
		return "\t\tip saddr @lan_subnets4 jump from_lan\n"
	}
	return fmt.Sprintf("\t\tiifname %s ip saddr @lan_subnets4 jump from_lan\n", ifaceMatcher(cfg.Main.LANIfaces))
}

func ifaceMatcher(ifaces []string) string {
	quoted := make([]string, 0, len(ifaces))
	for _, iface := range ifaces {
		quoted = append(quoted, quoteNFTString(iface))
	}
	if len(quoted) == 1 {
		return quoted[0]
	}
	return "{ " + strings.Join(quoted, ", ") + " }"
}

func quoteNFTString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

func counter(cfg config.Config) string {
	if cfg.Main.NFTCounters {
		return " counter"
	}
	return ""
}

func writeSet(b *strings.Builder, name string, elements []string) {
	sort.Strings(elements)
	b.WriteString(fmt.Sprintf("\tset %s {\n", name))
	b.WriteString("\t\ttype ipv4_addr\n")
	b.WriteString("\t\tflags interval\n")
	b.WriteString("\t\tauto-merge\n")
	if len(elements) > 0 {
		b.WriteString("\t\telements = {")
		for i, element := range elements {
			if i == 0 {
				b.WriteString(" ")
			} else {
				b.WriteString(", ")
			}
			b.WriteString(element)
		}
		b.WriteString(" }\n")
	}
	b.WriteString("\t}\n")
}

func collectClients(cfg config.Config, policy string) []string {
	var out []string
	for _, client := range cfg.Clients {
		if client.Policy == policy {
			out = append(out, client.IP)
		}
	}
	return out
}

func reservedCIDRs(cfg config.Config) []*net.IPNet {
	values := []string{
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"192.168.0.0/16",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"224.0.0.0/4",
		"240.0.0.0/4",
	}
	// 198.18.0.0/15 is intentionally omitted when used as FakeIP space.
	if !cfg.Main.FakeIPEnabled || cfg.Main.FakeIPRange != "198.18.0.0/15" {
		values = append(values, "198.18.0.0/15")
	}
	return policy.MustIPv4CIDRs(values...)
}

package nft

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/elllkere/neto/internal/config"
	"github.com/elllkere/neto/internal/policy"
	"github.com/elllkere/neto/internal/ruleengine"
)

type Input struct {
	Config    config.Config
	RuleCIDRs map[int][]*net.IPNet
}

func Generate(in Input) (string, error) {
	cfg := in.Config
	directClients := collectClients(cfg, "direct")
	proxyClients := collectClients(cfg, "proxy")
	lanSubnets := policy.NormalizeIPv4CIDRs(policy.MustIPv4CIDRs(cfg.Main.LANSubnets...))
	reserved4 := reservedCIDRs(cfg)

	var b strings.Builder
	b.WriteString("table inet neto {\n")
	writeSet(&b, "lan_subnets4", policy.CIDRStrings(lanSubnets))
	writeSet(&b, "reserved4", policy.CIDRStrings(reserved4))
	writeSet(&b, "direct_clients4", directClients)
	writeSet(&b, "proxy_clients4", proxyClients)
	writeRuleSets(&b, cfg, in.RuleCIDRs)
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
		if err := writeOrderedIPRules(&b, cfg, in.RuleCIDRs); err != nil {
			return "", err
		}
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

func writeRuleSets(b *strings.Builder, cfg config.Config, ruleCIDRs map[int][]*net.IPNet) {
	for i, rule := range cfg.Rules {
		if !ruleengine.HasIPMatch(rule) || len(ruleCIDRs[i]) == 0 {
			continue
		}
		writeSet(b, ruleSetName(i), policy.CIDRStrings(policy.NormalizeIPv4CIDRs(ruleCIDRs[i])))
	}
}

func writeOrderedIPRules(b *strings.Builder, cfg config.Config, ruleCIDRs map[int][]*net.IPNet) error {
	for i, rule := range cfg.Rules {
		if !ruleengine.HasIPMatch(rule) {
			continue
		}
		if len(ruleCIDRs[i]) == 0 {
			continue
		}
		matches, err := rulePacketMatches(ruleSetName(i), rule)
		if err != nil {
			return fmt.Errorf("rule %q packet match: %w", rule.Name, err)
		}
		switch rule.Action {
		case "proxy":
			for _, match := range matches {
				b.WriteString(fmt.Sprintf("\t\t%s%s jump to_proxy_default\n", match, counter(cfg)))
			}
		case "direct", "block":
			for _, match := range matches {
				b.WriteString(fmt.Sprintf("\t\t%s%s return\n", match, counter(cfg)))
			}
		}
	}
	return nil
}

func rulePacketMatches(setName string, rule config.Rule) ([]string, error) {
	base := "ip daddr @" + setName
	protos := packetProtocols(rule)
	srcPorts, err := parseNFTPorts(rule.SrcPorts)
	if err != nil {
		return nil, err
	}
	dstPorts, err := parseNFTPorts(rule.DstPorts)
	if err != nil {
		return nil, err
	}
	hasPorts := len(srcPorts)+len(dstPorts) > 0

	if !hasPorts {
		if len(protos) == 0 {
			return []string{base + " meta l4proto { tcp, udp }"}, nil
		}
		out := make([]string, 0, len(protos))
		for _, proto := range protos {
			out = append(out, base+" meta l4proto "+proto)
		}
		return out, nil
	}

	if len(protos) == 0 {
		protos = []string{"tcp", "udp"}
	}
	if len(srcPorts) == 0 {
		srcPorts = []string{""}
	}
	if len(dstPorts) == 0 {
		dstPorts = []string{""}
	}

	var out []string
	for _, proto := range protos {
		for _, srcPort := range srcPorts {
			for _, dstPort := range dstPorts {
				parts := []string{base}
				if srcPort != "" {
					parts = append(parts, proto+" sport "+srcPort)
				}
				if dstPort != "" {
					parts = append(parts, proto+" dport "+dstPort)
				}
				out = append(out, strings.Join(parts, " "))
			}
		}
	}
	return out, nil
}

func packetProtocols(rule config.Rule) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(proto string) {
		proto = strings.ToLower(strings.TrimSpace(proto))
		if proto != "tcp" && proto != "udp" {
			return
		}
		if _, ok := seen[proto]; ok {
			return
		}
		seen[proto] = struct{}{}
		out = append(out, proto)
	}
	for _, proto := range rule.Proto {
		add(proto)
	}
	if _, ok := seen["tcp"]; ok {
		if _, ok := seen["udp"]; ok {
			return []string{"tcp", "udp"}
		}
	}
	return out
}

func parseNFTPorts(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	for _, value := range values {
		port, err := parseNFTPort(value)
		if err != nil {
			return nil, err
		}
		out = append(out, port)
	}
	return out, nil
}

func parseNFTPort(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("empty port")
	}
	parts := strings.Split(value, "-")
	if len(parts) > 2 {
		return "", fmt.Errorf("invalid port range %q", value)
	}
	start, err := parsePortNumber(parts[0])
	if err != nil {
		return "", err
	}
	end := start
	if len(parts) == 2 {
		end, err = parsePortNumber(parts[1])
		if err != nil {
			return "", err
		}
		if start > end {
			return "", fmt.Errorf("invalid port range %q", value)
		}
		return strconv.Itoa(start) + "-" + strconv.Itoa(end), nil
	}
	return strconv.Itoa(start), nil
}

func parsePortNumber(value string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, err
	}
	if n < 1 || n > 65535 {
		return 0, fmt.Errorf("port must be 1..65535")
	}
	return n, nil
}

func ruleSetName(index int) string {
	return fmt.Sprintf("rule4_%04d", index)
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

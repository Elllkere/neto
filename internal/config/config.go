package config

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
)

const DefaultPath = "/etc/config/neto"

type Config struct {
	Main     Main
	Clients  []Client
	Rules    []Rule
	Warnings []string
}

type Main struct {
	Enabled               bool
	DNSListen             string
	RealDNSUpstream       string
	ManageDNSMasq         bool
	FilterAAAAForFakeIP   bool
	SingBoxBin            string
	SingBoxDNS            string
	TProxyPort            int
	Mark                  string
	Table                 int
	FakeIPEnabled         bool
	FakeIPRange           string
	ResolveForSubnetRules bool
	NFTCounters           bool
	DefaultAction         string
	RoutingMode           string
	DefaultOutbound       string
	LANSubnets            []string
	LANIfaces             []string
}

type Client struct {
	Name   string
	IP     string
	Policy string
}

type Rule struct {
	Name                    string
	Enabled                 bool
	Priority                int
	Action                  string
	Outbound                string
	DNSMode                 string
	DomainEquals            []string
	DomainContains          []string
	DomainStartsWith        []string
	DomainEndsWith          []string
	ExcludeDomainEquals     []string
	ExcludeDomainContains   []string
	ExcludeDomainStartsWith []string
	ExcludeDomainEndsWith   []string
	Files                   []string
}

type section struct {
	typ     string
	name    string
	options map[string]string
	lists   map[string][]string
}

func Defaults() Config {
	return Config{
		Main: Main{
			Enabled:               true,
			DNSListen:             "127.0.0.1:5353",
			RealDNSUpstream:       "1.1.1.1:53",
			ManageDNSMasq:         true,
			FilterAAAAForFakeIP:   true,
			SingBoxBin:            "/usr/libexec/neto/sing-box",
			SingBoxDNS:            "127.0.0.1:15353",
			TProxyPort:            16001,
			Mark:                  "0x101",
			Table:                 101,
			FakeIPEnabled:         true,
			FakeIPRange:           "198.18.0.0/15",
			ResolveForSubnetRules: true,
			NFTCounters:           true,
			DefaultAction:         "direct",
			RoutingMode:           "custom",
			DefaultOutbound:       "direct",
			LANSubnets:            []string{"192.168.8.0/24"},
		},
	}
}

func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	return Parse(string(data))
}

func Parse(data string) (Config, error) {
	sections, err := parseSections(data)
	if err != nil {
		return Config{}, err
	}

	cfg := Defaults()
	for _, s := range sections {
		switch s.typ {
		case "main":
			applyMain(&cfg.Main, s)
		case "client":
			rawPolicy := firstNonEmpty(s.options["policy"], "default")
			policy, warning := normalizeClientPolicy(rawPolicy)
			if warning != "" {
				cfg.Warnings = append(cfg.Warnings, warning)
			}
			cfg.Clients = append(cfg.Clients, Client{
				Name:   firstNonEmpty(s.options["name"], s.name),
				IP:     s.options["ip"],
				Policy: policy,
			})
		case "rule":
			rule, warning := parseRule(s, len(cfg.Rules))
			cfg.Rules = append(cfg.Rules, rule)
			if warning != "" {
				cfg.Warnings = append(cfg.Warnings, warning)
			}
		}
	}
	sort.SliceStable(cfg.Rules, func(i, j int) bool {
		return cfg.Rules[i].Priority < cfg.Rules[j].Priority
	})

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	switch c.Main.RoutingMode {
	case "custom", "global":
	default:
		return fmt.Errorf("unsupported routing_mode %q", c.Main.RoutingMode)
	}
	if c.Main.DefaultOutbound == "" {
		return fmt.Errorf("default_outbound must not be empty")
	}
	if c.Main.DefaultOutbound != "direct" {
		return fmt.Errorf("unsupported default_outbound %q: only direct is supported in v1", c.Main.DefaultOutbound)
	}
	if len(c.Main.LANSubnets) == 0 {
		return fmt.Errorf("at least one lan_subnet is required")
	}
	for _, cidr := range c.Main.LANSubnets {
		if ip, _, err := net.ParseCIDR(cidr); err != nil || ip.To4() == nil {
			if err == nil {
				err = fmt.Errorf("not an IPv4 CIDR")
			}
			return fmt.Errorf("invalid lan_subnet %q: %w", cidr, err)
		}
	}
	for _, iface := range c.Main.LANIfaces {
		if strings.TrimSpace(iface) == "" {
			return fmt.Errorf("lan_iface must not be empty")
		}
	}
	if _, _, err := net.ParseCIDR(c.Main.FakeIPRange); c.Main.FakeIPEnabled && err != nil {
		return fmt.Errorf("invalid fakeip_range %q: %w", c.Main.FakeIPRange, err)
	}
	if c.Main.TProxyPort <= 0 || c.Main.TProxyPort > 65535 {
		return fmt.Errorf("invalid tproxy_port %d", c.Main.TProxyPort)
	}
	if c.Main.Table <= 0 {
		return fmt.Errorf("invalid routing table %d", c.Main.Table)
	}
	if strings.TrimSpace(c.Main.Mark) == "" {
		return fmt.Errorf("mark must not be empty")
	}
	if strings.TrimSpace(c.Main.SingBoxBin) == "" {
		return fmt.Errorf("singbox_bin must not be empty")
	}
	if _, _, err := net.SplitHostPort(c.Main.SingBoxDNS); err != nil {
		return fmt.Errorf("invalid singbox_dns %q: %w", c.Main.SingBoxDNS, err)
	}
	if _, _, err := net.SplitHostPort(c.Main.DNSListen); err != nil {
		return fmt.Errorf("invalid dns_listen %q: %w", c.Main.DNSListen, err)
	}
	if _, _, err := net.SplitHostPort(c.Main.RealDNSUpstream); err != nil {
		return fmt.Errorf("invalid real_dns_upstream %q: %w", c.Main.RealDNSUpstream, err)
	}
	if c.Main.RealDNSUpstream == c.Main.DNSListen {
		return fmt.Errorf("real_dns_upstream must not point back to dns_listen")
	}
	if c.Main.SingBoxDNS == c.Main.DNSListen {
		return fmt.Errorf("singbox_dns must not point back to dns_listen")
	}

	for i, cl := range c.Clients {
		if net.ParseIP(cl.IP).To4() == nil {
			return fmt.Errorf("client %d has invalid IPv4 address %q", i, cl.IP)
		}
		switch cl.Policy {
		case "default", "proxy", "direct":
		default:
			return fmt.Errorf("client %q has unsupported policy %q", cl.Name, cl.Policy)
		}
	}
	for _, r := range c.Rules {
		if strings.TrimSpace(r.Name) == "" {
			return fmt.Errorf("rule name must not be empty")
		}
		switch r.Action {
		case "proxy", "direct", "block":
		default:
			return fmt.Errorf("rule %q has unsupported action %q", r.Name, r.Action)
		}
		switch r.DNSMode {
		case "fakeip", "real_ip", "auto":
		default:
			return fmt.Errorf("rule %q has unsupported dns_mode %q", r.Name, r.DNSMode)
		}
		if r.Outbound != "" && r.Outbound != "proxy_default" {
			return fmt.Errorf("rule %q has unsupported outbound %q", r.Name, r.Outbound)
		}
		for _, file := range r.Files {
			if strings.TrimSpace(file) == "" {
				return fmt.Errorf("rule %q contains an empty provider file path", r.Name)
			}
		}
	}
	return nil
}

func applyMain(m *Main, s section) {
	if v, ok := s.options["enabled"]; ok {
		m.Enabled = parseBool(v, m.Enabled)
	}
	if v := s.options["dns_listen"]; v != "" {
		m.DNSListen = v
	}
	if v := s.options["real_dns_upstream"]; v != "" {
		m.RealDNSUpstream = v
	}
	if v, ok := s.options["manage_dnsmasq"]; ok {
		m.ManageDNSMasq = parseBool(v, m.ManageDNSMasq)
	}
	if v, ok := s.options["filter_aaaa_for_fakeip"]; ok {
		m.FilterAAAAForFakeIP = parseBool(v, m.FilterAAAAForFakeIP)
	}
	if v := s.options["singbox_bin"]; v != "" {
		m.SingBoxBin = v
	}
	if v := s.options["singbox_dns"]; v != "" {
		m.SingBoxDNS = v
	}
	if v := s.options["tproxy_port"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			m.TProxyPort = n
		}
	}
	if v := s.options["mark"]; v != "" {
		m.Mark = v
	}
	if v := s.options["table"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			m.Table = n
		}
	}
	if v, ok := s.options["fakeip_enabled"]; ok {
		m.FakeIPEnabled = parseBool(v, m.FakeIPEnabled)
	}
	if v := s.options["fakeip_range"]; v != "" {
		m.FakeIPRange = v
	}
	if v, ok := s.options["resolve_for_subnet_rules"]; ok {
		m.ResolveForSubnetRules = parseBool(v, m.ResolveForSubnetRules)
	}
	if v, ok := s.options["nft_counters"]; ok {
		m.NFTCounters = parseBool(v, m.NFTCounters)
	}
	if v := s.options["routing_mode"]; v != "" {
		m.RoutingMode = v
	}
	if v := s.options["default_outbound"]; v != "" {
		m.DefaultOutbound = v
	}
	if v := s.options["default_action"]; v != "" {
		m.DefaultAction = v
	}
	if values, ok := s.lists["lan_subnet"]; ok {
		m.LANSubnets = cleanList(values)
	}
	if values, ok := s.lists["lan_iface"]; ok {
		m.LANIfaces = cleanList(values)
	}
}

func parseRule(s section, order int) (Rule, string) {
	r := Rule{
		Name:     firstNonEmpty(s.options["name"], s.name, fmt.Sprintf("rule_%d", order+1)),
		Enabled:  true,
		Priority: 1000 + order,
		Action:   firstNonEmpty(s.options["action"], "proxy"),
		Outbound: firstNonEmpty(s.options["outbound"], "proxy_default"),
		DNSMode:  firstNonEmpty(s.options["dns_mode"], "auto"),
	}
	if v, ok := s.options["enabled"]; ok {
		r.Enabled = parseBool(v, r.Enabled)
	}
	if v := s.options["priority"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			r.Priority = n
		}
	}
	var warning string
	if _, ok := s.options["match_all"]; ok {
		warning = fmt.Sprintf("rule %q: match_all is deprecated and ignored", r.Name)
	}
	r.DomainEquals = cleanDomainList(appendList(s.lists["domain_equals"], s.lists["domain_exact"]))
	r.DomainContains = cleanDomainList(appendList(s.lists["domain_contains"], s.lists["domain_keyword"]))
	r.DomainStartsWith = cleanDomainList(appendList(s.lists["domain_starts_with"], s.lists["domain_prefix"]))
	r.DomainEndsWith = cleanDomainList(appendList(s.lists["domain_ends_with"], s.lists["domain_suffix"]))
	r.ExcludeDomainEquals = cleanDomainList(appendList(s.lists["exclude_domain_equals"], s.lists["exclude_domain_exact"]))
	r.ExcludeDomainContains = cleanDomainList(appendList(s.lists["exclude_domain_contains"], s.lists["exclude_domain_keyword"]))
	r.ExcludeDomainStartsWith = cleanDomainList(appendList(s.lists["exclude_domain_starts_with"], s.lists["exclude_domain_prefix"]))
	r.ExcludeDomainEndsWith = cleanDomainList(appendList(s.lists["exclude_domain_ends_with"], s.lists["exclude_domain_suffix"]))
	r.Files = cleanList(s.lists["file"])
	return r, warning
}

func cleanList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func appendList(a []string, b []string) []string {
	out := make([]string, 0, len(a)+len(b))
	out = append(out, a...)
	out = append(out, b...)
	return out
}

func cleanDomainList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimRight(strings.ToLower(strings.TrimSpace(value)), ".")
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeClientPolicy(policy string) (string, string) {
	switch strings.TrimSpace(policy) {
	case "", "default":
		return "default", ""
	case "proxy":
		return "proxy", ""
	case "proxy_default":
		return "proxy", "policy 'proxy_default' is deprecated; using 'proxy'"
	case "direct":
		return "direct", ""
	case "bypass":
		return "direct", "policy 'bypass' is deprecated; using 'direct'"
	default:
		return policy, ""
	}
}

func parseSections(data string) ([]section, error) {
	var sections []section
	var cur *section

	for lineNo, raw := range strings.Split(data, "\n") {
		line := strings.TrimSpace(stripComment(raw))
		if line == "" {
			continue
		}
		fields, err := splitUCIFields(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo+1, err)
		}
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "config":
			if len(fields) < 2 || len(fields) > 3 {
				return nil, fmt.Errorf("line %d: invalid config statement", lineNo+1)
			}
			sections = append(sections, section{
				typ:     fields[1],
				options: map[string]string{},
				lists:   map[string][]string{},
			})
			cur = &sections[len(sections)-1]
			if len(fields) == 3 {
				cur.name = fields[2]
			}
		case "option":
			if cur == nil {
				return nil, fmt.Errorf("line %d: option outside config section", lineNo+1)
			}
			if len(fields) != 3 {
				return nil, fmt.Errorf("line %d: invalid option statement", lineNo+1)
			}
			cur.options[fields[1]] = fields[2]
		case "list":
			if cur == nil {
				return nil, fmt.Errorf("line %d: list outside config section", lineNo+1)
			}
			if len(fields) != 3 {
				return nil, fmt.Errorf("line %d: invalid list statement", lineNo+1)
			}
			cur.lists[fields[1]] = append(cur.lists[fields[1]], fields[2])
		default:
			return nil, fmt.Errorf("line %d: unsupported UCI statement %q", lineNo+1, fields[0])
		}
	}

	return sections, nil
}

func stripComment(s string) string {
	var quote rune
	for i, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '\'' || r == '"':
			quote = r
		case r == '#':
			return s[:i]
		}
	}
	return s
}

func splitUCIFields(s string) ([]string, error) {
	var fields []string
	var b strings.Builder
	var quote rune
	inField := false

	flush := func() {
		if inField {
			fields = append(fields, b.String())
			b.Reset()
			inField = false
		}
	}

	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			b.WriteRune(r)
			inField = true
		case r == '\'' || r == '"':
			quote = r
			inField = true
		case r == ' ' || r == '\t':
			flush()
		default:
			b.WriteRune(r)
			inField = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	flush()
	return fields, nil
}

func parseBool(v string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on", "enabled":
		return true
	case "0", "false", "no", "off", "disabled":
		return false
	default:
		return fallback
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

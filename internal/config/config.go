package config

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/elllkere/neto/internal/policy"
)

const DefaultPath = "/etc/config/neto"

var ProviderCacheDir = "/etc/neto/provider-cache"
var ProviderPersistentCacheDir = "/etc/neto/provider-cache"
var ProviderLegacyCacheDir = "/var/lib/neto/providers"

const (
	BuiltinDirectOutbound  = "direct"
	BuiltinBlockedOutbound = "blocked"
)

type Config struct {
	Main          Main
	Clients       []Client
	Rules         []Rule
	Providers     []Provider
	Outbounds     []Outbound
	Subscriptions []Subscription
	Warnings      []string
}

type Main struct {
	Enabled               bool
	DNSListen             string
	RealDNSUpstream       string
	RealDNSMode           string
	RealDNSOutbound       string
	RealDNSTransport      string
	RealDNSServer         string
	RealDNSServerPort     int
	RealDNSServerName     string
	RealDNSPath           string
	DNSUpstreamPreset     string
	DNSUpstreamProtocol   string
	DNSUpstreamHost       string
	DNSUpstreamPort       int
	DNSUpstreamTLSName    string
	DNSUpstreamPath       string
	ManageDNSMasq         bool
	FilterAAAAForFakeIP   bool
	SingBoxBin            string
	SingBoxDNS            string
	SingBoxDNSFakeIP      string
	SingBoxDNSRealDirect  string
	SingBoxDNSRealProxy   string
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
	DomainFiles             []string
	IPCIDRs                 []string
	Files                   []string
	DomainProviders         []string
	IPProviders             []string
	Providers               []string
	Proto                   []string
	SrcPorts                []string
	DstPorts                []string
}

type Provider struct {
	Name           string
	Label          string
	Enabled        bool
	Type           string
	URL            string
	LocalPath      string
	AutoUpdate     bool
	UpdateHour     int
	UpdateVia      string
	UpdateOutbound string
	LastUpdate     string
	ItemCount      int
	Files          []string
}

func (p Provider) CachePath() string {
	localPath := strings.TrimSpace(p.LocalPath)
	if localPath != "" && !p.isDefaultCachePath(localPath) {
		return localPath
	}
	return p.DefaultCachePath()
}

func (p Provider) DefaultCachePath() string {
	return providerCachePath(ProviderCacheDir, p.Name)
}

func (p Provider) LegacyCachePath() string {
	return providerCachePath(ProviderLegacyCacheDir, p.Name)
}

func providerCachePath(dir string, providerName string) string {
	providerName = safeProviderName(providerName)
	if providerName == "" {
		providerName = "provider"
	}
	return filepath.Join(dir, providerName+".txt")
}

func (p Provider) PersistentCachePath() string {
	return providerCachePath(ProviderPersistentCacheDir, p.Name)
}

func (p Provider) UsesDefaultCachePath() bool {
	localPath := strings.TrimSpace(p.LocalPath)
	if localPath == "" {
		return true
	}
	return p.isDefaultCachePath(localPath)
}

func (p Provider) isDefaultCachePath(path string) bool {
	clean := filepath.Clean(path)
	return clean == filepath.Clean(p.DefaultCachePath()) ||
		clean == filepath.Clean(p.PersistentCachePath()) ||
		clean == filepath.Clean(p.LegacyCachePath())
}

func (p Provider) RestoreDefaultCache() (bool, error) {
	if !p.UsesDefaultCachePath() {
		return false, nil
	}
	data, err := os.ReadFile(p.PersistentCachePath())
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	cachePath := p.CachePath()
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return false, err
	}
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return false, err
	}
	return true, nil
}

func (p Provider) MirrorDefaultCache() error {
	if !p.UsesDefaultCachePath() {
		return nil
	}
	data, err := os.ReadFile(p.CachePath())
	if err != nil {
		return err
	}
	return p.WritePersistentCache(data)
}

func (p Provider) WritePersistentCache(data []byte) error {
	if !p.UsesDefaultCachePath() {
		return nil
	}
	path := p.PersistentCachePath()
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, data) {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (c Config) ProviderByName(name string) (Provider, bool) {
	name = strings.TrimSpace(name)
	for _, provider := range c.Providers {
		if provider.Name == name {
			return provider, true
		}
	}
	return Provider{}, false
}

type Outbound struct {
	Enabled              bool
	Tag                  string
	Label                string
	Type                 string
	Server               string
	Port                 int
	UUID                 string
	Flow                 string
	TLS                  bool
	ServerName           string
	Reality              bool
	RealityPublicKey     string
	RealityShortID       string
	ALPN                 []string
	TLSMinVersion        string
	TLSMaxVersion        string
	TLSCipherSuites      []string
	ECH                  bool
	ECHConfig            []string
	ECHConfigPath        string
	UTLSFingerprint      string
	Transport            string
	PacketEncoding       string
	GRPCServiceName      string
	HTTPHost             []string
	HTTPUpgradeHost      string
	HTTPPath             string
	HTTPMethod           string
	WSHost               string
	WSPath               string
	WSEarlyData          int
	WSEarlyDataHeader    string
	Password             string
	Method               string
	Insecure             bool
	HysteriaObfsType     string
	HysteriaObfsPassword string
	HysteriaUpMbps       int
	HysteriaDownMbps     int
}

type Subscription struct {
	Name           string
	Label          string
	Enabled        bool
	URL            string
	AutoUpdate     bool
	UpdateHour     int
	UpdateInterval string
	UpdateVia      string
	UpdateOutbound string
	LastUpdate     string
	NodeCount      int
}

type DNSUpstream struct {
	Preset   string
	Protocol string
	Host     string
	Port     int
	TLSName  string
	Path     string
}

type section struct {
	typ     string
	name    string
	options map[string]string
	lists   map[string][]string
}

func BuiltinOutboundTags() []string {
	return []string{BuiltinDirectOutbound, BuiltinBlockedOutbound}
}

func Defaults() Config {
	return Config{
		Main: Main{
			Enabled:               true,
			DNSListen:             "127.0.0.1:5353",
			RealDNSUpstream:       "1.1.1.1:53",
			RealDNSMode:           "direct",
			RealDNSTransport:      "udp",
			RealDNSServer:         "1.1.1.1",
			RealDNSServerPort:     53,
			RealDNSServerName:     "cloudflare-dns.com",
			RealDNSPath:           "/dns-query",
			DNSUpstreamPreset:     "cloudflare",
			DNSUpstreamProtocol:   "udp",
			DNSUpstreamHost:       "1.1.1.1",
			DNSUpstreamPort:       53,
			DNSUpstreamTLSName:    "cloudflare-dns.com",
			DNSUpstreamPath:       "/dns-query",
			ManageDNSMasq:         true,
			FilterAAAAForFakeIP:   true,
			SingBoxBin:            "/usr/libexec/neto/sing-box",
			SingBoxDNS:            "127.0.0.1:15353",
			SingBoxDNSFakeIP:      "127.0.0.1:15353",
			SingBoxDNSRealDirect:  "127.0.0.1:15354",
			SingBoxDNSRealProxy:   "127.0.0.1:15355",
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

func (c Config) EnabledCustomOutbounds() []Outbound {
	outbounds := make([]Outbound, 0, len(c.Outbounds))
	for _, outbound := range c.Outbounds {
		if outbound.Enabled {
			outbounds = append(outbounds, outbound)
		}
	}
	return outbounds
}

func (c Config) AllowedOutboundTags() map[string]struct{} {
	tags := map[string]struct{}{
		BuiltinDirectOutbound:  {},
		BuiltinBlockedOutbound: {},
	}
	for _, outbound := range c.Outbounds {
		if outbound.Enabled && strings.TrimSpace(outbound.Tag) != "" {
			tags[outbound.Tag] = struct{}{}
		}
	}
	return tags
}

func (m Main) DNSUpstream() DNSUpstream {
	protocol := normalizeDNSProtocol(firstNonEmpty(m.RealDNSTransport, m.DNSUpstreamProtocol))
	if protocol == "" {
		protocol = "udp"
	}
	preset := strings.TrimSpace(firstNonEmpty(m.DNSUpstreamPreset, "custom"))
	if preset == "" {
		preset = "custom"
	}
	if preset != "custom" {
		host, tlsName, path := presetDNSUpstream(preset, protocol)
		return DNSUpstream{
			Preset:   preset,
			Protocol: protocol,
			Host:     host,
			Port:     defaultDNSPort(protocol),
			TLSName:  tlsName,
			Path:     path,
		}
	}

	host := strings.TrimSpace(m.RealDNSServer)
	port := m.RealDNSServerPort
	if h, p, ok := splitOptionalHostPort(host); ok {
		host = h
		if p > 0 {
			port = p
		}
	}
	u := DNSUpstream{
		Preset:   preset,
		Protocol: protocol,
		Host:     host,
		Port:     port,
		TLSName:  strings.TrimSpace(m.RealDNSServerName),
		Path:     strings.TrimSpace(m.RealDNSPath),
	}

	if u.Host == "" {
		legacy := legacyDNSUpstream(m)
		u.Host = legacy.Host
		u.Port = legacy.Port
		u.TLSName = legacy.TLSName
		u.Path = legacy.Path
		u.Preset = legacy.Preset
	}
	if u.Host == "" {
		host, port, ok := splitHostPortValue(m.RealDNSUpstream)
		if ok {
			u.Host = host
			if u.Port == 0 {
				u.Port = port
			}
		}
	}
	if u.Port == 0 {
		u.Port = defaultDNSPort(protocol)
	}
	if protocol == "https" && u.Path == "" {
		u.Path = "/dns-query"
	}
	return u
}

func (m Main) SingBoxDNSFakeIPAddr() string {
	return firstNonEmpty(strings.TrimSpace(m.SingBoxDNSFakeIP), strings.TrimSpace(m.SingBoxDNS), "127.0.0.1:15353")
}

func (m Main) SingBoxDNSRealDirectAddr() string {
	return firstNonEmpty(strings.TrimSpace(m.SingBoxDNSRealDirect), "127.0.0.1:15354")
}

func (m Main) SingBoxDNSRealProxyAddr() string {
	return firstNonEmpty(strings.TrimSpace(m.SingBoxDNSRealProxy), "127.0.0.1:15355")
}

func legacyDNSUpstream(m Main) DNSUpstream {
	protocol := normalizeDNSProtocol(m.DNSUpstreamProtocol)
	if protocol == "" {
		protocol = "udp"
	}
	preset := strings.TrimSpace(m.DNSUpstreamPreset)
	if preset == "" {
		preset = "custom"
	}

	u := DNSUpstream{
		Preset:   preset,
		Protocol: protocol,
		Host:     strings.TrimSpace(m.DNSUpstreamHost),
		Port:     m.DNSUpstreamPort,
		TLSName:  strings.TrimSpace(m.DNSUpstreamTLSName),
		Path:     strings.TrimSpace(m.DNSUpstreamPath),
	}

	if preset != "custom" {
		u.Host, u.TLSName, u.Path = presetDNSUpstream(preset, protocol)
		u.Port = defaultDNSPort(protocol)
	}
	if u.Host == "" {
		host, port, ok := splitHostPortValue(m.RealDNSUpstream)
		if ok {
			u.Host = host
			if u.Port == 0 {
				u.Port = port
			}
		}
	}
	if u.Port == 0 {
		u.Port = defaultDNSPort(protocol)
	}
	if protocol == "https" && u.Path == "" {
		u.Path = "/dns-query"
	}
	return u
}

func (u DNSUpstream) Address() string {
	return net.JoinHostPort(u.Host, strconv.Itoa(u.Port))
}

func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	cfg, err := Parse(string(data))
	if err != nil {
		return Config{}, err
	}
	if err := LoadRuleDomainFiles(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Parse(data string) (Config, error) {
	sections, err := parseSections(data)
	if err != nil {
		return Config{}, err
	}

	cfg := Defaults()
	hasOutboundSection := false
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
			rule, warnings := parseRule(s, len(cfg.Rules))
			cfg.Rules = append(cfg.Rules, rule)
			cfg.Warnings = append(cfg.Warnings, warnings...)
		case "provider":
			cfg.Providers = append(cfg.Providers, parseProvider(s))
		case "outbound":
			if !hasOutboundSection {
				cfg.Outbounds = nil
				hasOutboundSection = true
			}
			outbound, warnings := parseOutbound(s)
			cfg.Outbounds = append(cfg.Outbounds, outbound)
			cfg.Warnings = append(cfg.Warnings, warnings...)
		case "subscription":
			cfg.Subscriptions = append(cfg.Subscriptions, parseSubscription(s))
		}
	}
	sort.SliceStable(cfg.Rules, func(i, j int) bool {
		return cfg.Rules[i].Priority < cfg.Rules[j].Priority
	})
	cfg.Main.LANSubnets = normalizeIPv4CIDRStrings(cfg.Main.LANSubnets)

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if !c.Main.Enabled {
		return nil
	}

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
	if _, _, err := net.SplitHostPort(c.Main.DNSListen); err != nil {
		return fmt.Errorf("invalid dns_listen %q: %w", c.Main.DNSListen, err)
	}
	dnsListeners := map[string]string{
		"singbox_dns_fakeip":      c.Main.SingBoxDNSFakeIPAddr(),
		"singbox_dns_real_direct": c.Main.SingBoxDNSRealDirectAddr(),
		"singbox_dns_real_proxy":  c.Main.SingBoxDNSRealProxyAddr(),
	}
	seenDNSListeners := map[string]string{
		c.Main.DNSListen: "dns_listen",
	}
	for name, addr := range dnsListeners {
		if _, _, err := net.SplitHostPort(addr); err != nil {
			return fmt.Errorf("invalid %s %q: %w", name, addr, err)
		}
		if previous := seenDNSListeners[addr]; previous != "" {
			return fmt.Errorf("%s must not duplicate %s", name, previous)
		}
		seenDNSListeners[addr] = name
	}
	if c.Main.RealDNSUpstream != "" {
		if _, _, err := net.SplitHostPort(c.Main.RealDNSUpstream); err != nil {
			return fmt.Errorf("invalid real_dns_upstream %q: %w", c.Main.RealDNSUpstream, err)
		}
	}
	switch c.Main.RealDNSMode {
	case "direct", "proxy":
	default:
		return fmt.Errorf("unsupported real_dns_mode %q", c.Main.RealDNSMode)
	}
	dnsUpstream := c.Main.DNSUpstream()
	switch dnsUpstream.Protocol {
	case "udp", "tcp", "tls", "https":
	default:
		return fmt.Errorf("unsupported real_dns_transport %q", firstNonEmpty(c.Main.RealDNSTransport, c.Main.DNSUpstreamProtocol))
	}
	switch dnsUpstream.Preset {
	case "cloudflare", "google", "custom":
	default:
		return fmt.Errorf("unsupported dns_upstream_preset %q", dnsUpstream.Preset)
	}
	if strings.TrimSpace(dnsUpstream.Host) == "" {
		return fmt.Errorf("dns upstream host must not be empty")
	}
	if dnsUpstream.Port <= 0 || dnsUpstream.Port > 65535 {
		return fmt.Errorf("invalid dns upstream port %d", dnsUpstream.Port)
	}
	if dnsUpstream.Protocol == "https" && !strings.HasPrefix(dnsUpstream.Path, "/") {
		return fmt.Errorf("real_dns_path must start with /")
	}
	if err := c.validateRealDNSNoLoop(dnsUpstream); err != nil {
		return err
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
	seenOutboundTags := map[string]struct{}{
		BuiltinDirectOutbound:  {},
		BuiltinBlockedOutbound: {},
	}
	for _, outbound := range c.Outbounds {
		if !outbound.Enabled {
			continue
		}
		tag := strings.TrimSpace(outbound.Tag)
		if tag == "" {
			return fmt.Errorf("outbound tag must not be empty")
		}
		if reservedCustomOutboundTag(tag) {
			return fmt.Errorf("outbound tag %q is reserved", tag)
		}
		if _, ok := seenOutboundTags[tag]; ok {
			return fmt.Errorf("duplicate enabled outbound tag %q", tag)
		}
		seenOutboundTags[tag] = struct{}{}
		switch outbound.Type {
		case "vless":
			if err := requireOutboundFields(outbound, "server", "port", "uuid"); err != nil {
				return err
			}
			if outbound.Reality && strings.TrimSpace(outbound.RealityPublicKey) == "" {
				return fmt.Errorf("outbound %q reality_public_key is required when reality=1", outbound.Tag)
			}
		case "hysteria2":
			if err := requireOutboundFields(outbound, "server", "port", "password"); err != nil {
				return err
			}
		case "shadowsocks":
			if err := requireOutboundFields(outbound, "server", "port", "method", "password"); err != nil {
				return err
			}
		case "trojan":
			if err := requireOutboundFields(outbound, "server", "port", "password"); err != nil {
				return err
			}
		default:
			return fmt.Errorf("outbound %q has unsupported type %q", outbound.Tag, outbound.Type)
		}
	}
	if c.Main.RealDNSMode == "proxy" {
		realDNSOutbound := strings.TrimSpace(c.Main.RealDNSOutbound)
		if realDNSOutbound == "" {
			return fmt.Errorf("real_dns_mode=proxy requires real_dns_outbound")
		}
		if realDNSOutbound == "proxy_default" {
			return fmt.Errorf("real_dns_outbound 'proxy_default' is deprecated; choose a custom outbound")
		}
		if reservedCustomOutboundTag(realDNSOutbound) {
			return fmt.Errorf("real_dns_outbound %q must be a custom outbound", realDNSOutbound)
		}
		if _, ok := seenOutboundTags[realDNSOutbound]; !ok {
			return fmt.Errorf("real_dns_outbound %q not found", realDNSOutbound)
		}
	}
	seenSubscriptions := map[string]struct{}{}
	seenProviders := map[string]Provider{}
	for _, p := range c.Providers {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			return fmt.Errorf("provider name must not be empty")
		}
		if _, ok := seenProviders[name]; ok {
			return fmt.Errorf("duplicate provider %q", name)
		}
		seenProviders[name] = p
		if !p.Enabled {
			continue
		}
		switch p.Type {
		case "domain", "ip":
		default:
			return fmt.Errorf("provider %q has unsupported type %q", name, p.Type)
		}
		if strings.TrimSpace(p.URL) == "" && len(p.Files) == 0 && strings.TrimSpace(p.LocalPath) == "" {
			return fmt.Errorf("provider %q url is required", name)
		}
		switch p.UpdateVia {
		case "direct", "proxy":
		default:
			return fmt.Errorf("provider %q has unsupported update_via %q", name, p.UpdateVia)
		}
		if p.UpdateHour < 0 || p.UpdateHour > 23 {
			return fmt.Errorf("provider %q has invalid update_hour %d", name, p.UpdateHour)
		}
		if p.UpdateVia == "proxy" && strings.TrimSpace(p.UpdateOutbound) != "" {
			if _, ok := seenOutboundTags[p.UpdateOutbound]; !ok {
				return fmt.Errorf("provider %q has unsupported update_outbound %q", name, p.UpdateOutbound)
			}
			if p.UpdateOutbound == BuiltinBlockedOutbound {
				return fmt.Errorf("provider %q update_outbound must not be blocked", name)
			}
		}
	}
	for _, sub := range c.Subscriptions {
		name := strings.TrimSpace(sub.Name)
		if name == "" {
			return fmt.Errorf("subscription name must not be empty")
		}
		if _, ok := seenSubscriptions[name]; ok {
			return fmt.Errorf("duplicate subscription %q", name)
		}
		seenSubscriptions[name] = struct{}{}
		if !sub.Enabled {
			continue
		}
		if strings.TrimSpace(sub.URL) == "" {
			return fmt.Errorf("subscription %q url is required", name)
		}
		switch sub.UpdateVia {
		case "direct", "proxy":
		default:
			return fmt.Errorf("subscription %q has unsupported update_via %q", name, sub.UpdateVia)
		}
		if sub.UpdateHour < 0 || sub.UpdateHour > 23 {
			return fmt.Errorf("subscription %q has invalid update_hour %d", name, sub.UpdateHour)
		}
		if sub.UpdateVia == "proxy" && strings.TrimSpace(sub.UpdateOutbound) != "" {
			if _, ok := seenOutboundTags[sub.UpdateOutbound]; !ok {
				return fmt.Errorf("subscription %q has unsupported update_outbound %q", name, sub.UpdateOutbound)
			}
			if sub.UpdateOutbound == BuiltinBlockedOutbound {
				return fmt.Errorf("subscription %q update_outbound must not be blocked", name)
			}
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
		hasDomainSelectors := ruleHasDomainSelectors(r, seenProviders)
		hasIPSelectors := ruleHasIPSelectors(r, seenProviders)
		hasProtoSelectors := len(r.Proto) > 0
		hasPortSelectors := ruleHasPortSelectors(r)
		for _, proto := range r.Proto {
			switch strings.ToLower(strings.TrimSpace(proto)) {
			case "tcp", "udp":
			default:
				return fmt.Errorf("rule %q has unsupported proto %q", r.Name, proto)
			}
		}
		for _, port := range appendList(r.SrcPorts, r.DstPorts) {
			if _, _, err := parseRulePortRange(port); err != nil {
				return fmt.Errorf("rule %q contains invalid port match %q: %w", r.Name, port, err)
			}
		}
		if hasPortSelectors && !hasIPSelectors {
			return fmt.Errorf("rule %q: Port matching is packet-level and requires provider/CIDR/IP matchers in v1", r.Name)
		}
		if hasProtoSelectors && !hasIPSelectors {
			return fmt.Errorf("rule %q: Protocol matching is packet-level and requires provider/CIDR/IP matchers in v1", r.Name)
		}
		if hasIPSelectors && !hasDomainSelectors && r.DNSMode == "fakeip" {
			return fmt.Errorf("rule %q: provider/CIDR rules require real DNS because FakeIP hides destination IP", r.Name)
		}
		if r.Action == "direct" && r.DNSMode == "fakeip" {
			return fmt.Errorf("rule %q: direct rules require real DNS", r.Name)
		}
		outboundTag := strings.TrimSpace(firstNonEmpty(r.Outbound, BuiltinDirectOutbound))
		if _, ok := seenOutboundTags[outboundTag]; !ok {
			return fmt.Errorf("rule %q has unsupported outbound %q", r.Name, outboundTag)
		}
		for _, file := range r.Files {
			if strings.TrimSpace(file) == "" {
				return fmt.Errorf("rule %q contains an empty provider file path", r.Name)
			}
		}
		for _, file := range r.DomainFiles {
			if strings.TrimSpace(file) == "" {
				return fmt.Errorf("rule %q contains an empty domain file path", r.Name)
			}
		}
		for _, cidr := range r.IPCIDRs {
			if _, err := policy.ParseIPv4CIDR(cidr); err != nil {
				return fmt.Errorf("rule %q contains invalid IPv4 CIDR %q: %w", r.Name, cidr, err)
			}
		}
		for _, providerName := range appendList(appendList(r.DomainProviders, r.IPProviders), r.Providers) {
			providerName = strings.TrimSpace(providerName)
			p, ok := seenProviders[providerName]
			if !ok {
				return fmt.Errorf("rule %q references unknown provider %q", r.Name, providerName)
			}
			if !p.Enabled {
				return fmt.Errorf("rule %q references disabled provider %q", r.Name, providerName)
			}
		}
	}
	return nil
}

func normalizeIPv4CIDRStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		ipnet, err := policy.ParseIPv4CIDR(value)
		if err != nil {
			return values
		}
		normalized := ipnet.String()
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func (c Config) validateRealDNSNoLoop(upstream DNSUpstream) error {
	host := strings.Trim(strings.TrimSpace(upstream.Host), "[]")
	hostLower := strings.ToLower(host)
	port := upstream.Port

	if hostLower == "localhost" {
		return fmt.Errorf("real_dns_server must not point to localhost")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() {
			return fmt.Errorf("real_dns_server must not point to loopback address %s", host)
		}
		if routerLANIP(c.Main.LANSubnets, ip) {
			return fmt.Errorf("real_dns_server must not point to router LAN IP %s", host)
		}
	}

	for name, addr := range map[string]string{
		"dns_listen":               c.Main.DNSListen,
		"singbox_dns_fakeip":       c.Main.SingBoxDNSFakeIPAddr(),
		"singbox_dns_real_direct":  c.Main.SingBoxDNSRealDirectAddr(),
		"singbox_dns_real_proxy":   c.Main.SingBoxDNSRealProxyAddr(),
		"dnsmasq loopback default": "127.0.0.1:53",
	} {
		addrHost, addrPortValue, err := net.SplitHostPort(addr)
		if err != nil {
			continue
		}
		addrPort, err := strconv.Atoi(addrPortValue)
		if err != nil {
			continue
		}
		if port == addrPort && sameHost(host, addrHost) {
			return fmt.Errorf("real_dns_server must not point back to %s", name)
		}
	}
	return nil
}

func applyMain(m *Main, s section) {
	hasDNSUpstreamPreset := false
	hasDNSUpstreamFields := false
	hasRealDNSFields := false
	hasRealDNSPort := false
	hasRealDNSUpstream := false
	if v, ok := s.options["enabled"]; ok {
		m.Enabled = parseBool(v, m.Enabled)
	}
	if v := s.options["dns_listen"]; v != "" {
		m.DNSListen = v
	}
	if v, ok := s.options["real_dns_upstream"]; ok && v != "" {
		m.RealDNSUpstream = v
		hasRealDNSUpstream = true
	}
	if v := s.options["real_dns_mode"]; v != "" {
		m.RealDNSMode = strings.ToLower(strings.TrimSpace(v))
	}
	if v, ok := s.options["real_dns_outbound"]; ok {
		m.RealDNSOutbound = strings.TrimSpace(v)
	}
	if v := s.options["real_dns_transport"]; v != "" {
		m.RealDNSTransport = normalizeDNSProtocol(v)
		hasRealDNSFields = true
	}
	if v := s.options["real_dns_server"]; v != "" {
		m.RealDNSServer = strings.TrimSpace(v)
		hasRealDNSFields = true
	}
	if v := s.options["real_dns_port"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			m.RealDNSServerPort = n
			hasRealDNSFields = true
			hasRealDNSPort = true
		}
	}
	if v := s.options["real_dns_server_name"]; v != "" {
		m.RealDNSServerName = strings.TrimSpace(v)
		hasRealDNSFields = true
	}
	if v := s.options["real_dns_path"]; v != "" {
		m.RealDNSPath = strings.TrimSpace(v)
		hasRealDNSFields = true
	}
	if v := s.options["dns_upstream_preset"]; v != "" {
		m.DNSUpstreamPreset = v
		hasDNSUpstreamPreset = true
		hasDNSUpstreamFields = true
	}
	if v := s.options["dns_upstream_protocol"]; v != "" {
		m.DNSUpstreamProtocol = normalizeDNSProtocol(v)
		hasDNSUpstreamFields = true
	}
	if v := s.options["dns_upstream_host"]; v != "" {
		m.DNSUpstreamHost = v
		hasDNSUpstreamFields = true
	}
	if v := s.options["dns_upstream_port"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			m.DNSUpstreamPort = n
			hasDNSUpstreamFields = true
		}
	}
	if v := s.options["dns_upstream_tls_name"]; v != "" {
		m.DNSUpstreamTLSName = v
		hasDNSUpstreamFields = true
	}
	if v := s.options["dns_upstream_path"]; v != "" {
		m.DNSUpstreamPath = v
		hasDNSUpstreamFields = true
	}
	if !hasRealDNSFields && (hasDNSUpstreamPreset || hasDNSUpstreamFields) {
		upstream := legacyDNSUpstream(*m)
		m.RealDNSTransport = upstream.Protocol
		m.RealDNSServer = upstream.Host
		m.RealDNSServerPort = upstream.Port
		m.RealDNSServerName = upstream.TLSName
		m.RealDNSPath = upstream.Path
	} else if !hasRealDNSFields && hasRealDNSUpstream {
		host, port, ok := splitHostPortValue(m.RealDNSUpstream)
		if ok {
			m.RealDNSTransport = "udp"
			m.RealDNSServer = host
			m.RealDNSServerPort = port
			m.RealDNSServerName = ""
			m.RealDNSPath = "/dns-query"
			m.DNSUpstreamPreset = "custom"
			m.DNSUpstreamHost = host
			m.DNSUpstreamPort = port
			m.DNSUpstreamProtocol = "udp"
		}
	}
	if hasRealDNSFields && !hasRealDNSPort {
		if _, _, ok := splitOptionalHostPort(m.RealDNSServer); !ok {
			m.RealDNSServerPort = defaultDNSPort(m.RealDNSTransport)
		}
	}
	if hasRealDNSFields && !hasDNSUpstreamPreset {
		m.DNSUpstreamPreset = "custom"
	}
	syncLegacyDNSFields(m)
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
		if s.options["singbox_dns_fakeip"] == "" {
			m.SingBoxDNSFakeIP = v
		}
	}
	if v := s.options["singbox_dns_fakeip"]; v != "" {
		m.SingBoxDNSFakeIP = v
		m.SingBoxDNS = v
	}
	if v := s.options["singbox_dns_real_direct"]; v != "" {
		m.SingBoxDNSRealDirect = v
	}
	if v := s.options["singbox_dns_real_proxy"]; v != "" {
		m.SingBoxDNSRealProxy = v
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

func parseRule(s section, order int) (Rule, []string) {
	outbound := strings.TrimSpace(firstNonEmpty(s.options["outbound"], BuiltinDirectOutbound))
	var warnings []string
	if outbound == "proxy_default" {
		outbound = BuiltinDirectOutbound
		warnings = append(warnings, fmt.Sprintf("rule %q: outbound 'proxy_default' is deprecated; using 'direct'", firstNonEmpty(s.options["name"], s.name, fmt.Sprintf("rule_%d", order+1))))
	}
	r := Rule{
		Name:     firstNonEmpty(s.options["name"], s.name, fmt.Sprintf("rule_%d", order+1)),
		Enabled:  true,
		Priority: 1000 + order,
		Action:   firstNonEmpty(s.options["action"], "proxy"),
		Outbound: outbound,
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
	if _, ok := s.options["match_all"]; ok {
		warnings = append(warnings, fmt.Sprintf("rule %q: match_all is deprecated and ignored", r.Name))
	}
	r.DomainEquals = cleanDomainList(appendList(s.lists["domain_equals"], s.lists["domain_exact"]))
	r.DomainContains = cleanDomainList(appendList(s.lists["domain_contains"], s.lists["domain_keyword"]))
	r.DomainStartsWith = cleanDomainList(appendList(s.lists["domain_starts_with"], s.lists["domain_prefix"]))
	r.DomainEndsWith = cleanDomainList(appendList(s.lists["domain_ends_with"], s.lists["domain_suffix"]))
	r.ExcludeDomainEquals = cleanDomainList(appendList(s.lists["exclude_domain_equals"], s.lists["exclude_domain_exact"]))
	r.ExcludeDomainContains = cleanDomainList(appendList(s.lists["exclude_domain_contains"], s.lists["exclude_domain_keyword"]))
	r.ExcludeDomainStartsWith = cleanDomainList(appendList(s.lists["exclude_domain_starts_with"], s.lists["exclude_domain_prefix"]))
	r.ExcludeDomainEndsWith = cleanDomainList(appendList(s.lists["exclude_domain_ends_with"], s.lists["exclude_domain_suffix"]))
	r.DomainFiles = cleanList(appendList(s.lists["domain_file"], splitListOption(s.options["domain_file"])))
	r.IPCIDRs = cleanList(appendList(s.lists["ip_cidr"], splitListOption(s.options["ip_cidr"])))
	r.Files = cleanList(appendList(appendList(s.lists["ip_file"], s.lists["file"]), splitListOption(s.options["ip_file"])))
	r.DomainProviders = cleanList(appendList(s.lists["domain_provider"], splitListOption(s.options["domain_provider"])))
	r.IPProviders = cleanList(appendList(s.lists["ip_provider"], splitListOption(s.options["ip_provider"])))
	r.Providers = cleanList(appendList(s.lists["provider"], splitListOption(s.options["provider"])))
	r.Proto = cleanList(s.lists["proto"])
	for i := range r.Proto {
		r.Proto[i] = strings.ToLower(strings.TrimSpace(r.Proto[i]))
	}
	r.SrcPorts = cleanList(s.lists["src_port"])
	r.DstPorts = cleanList(s.lists["dst_port"])
	return r, warnings
}

func parseProvider(s section) Provider {
	name := strings.TrimSpace(firstNonEmpty(s.name, s.options["name"], s.options["label"]))
	p := Provider{
		Name:           safeProviderName(name),
		Label:          strings.TrimSpace(firstNonEmpty(s.options["label"], s.options["name"], name)),
		Enabled:        true,
		Type:           strings.TrimSpace(firstNonEmpty(s.options["type"], "ip")),
		URL:            strings.TrimSpace(s.options["url"]),
		LocalPath:      strings.TrimSpace(s.options["local_path"]),
		AutoUpdate:     false,
		UpdateVia:      strings.TrimSpace(firstNonEmpty(s.options["update_via"], "direct")),
		UpdateOutbound: strings.TrimSpace(s.options["update_outbound"]),
		LastUpdate:     strings.TrimSpace(s.options["last_update"]),
		Files:          cleanList(appendList(s.lists["file"], splitListOption(s.options["file"]))),
	}
	if v, ok := s.options["enabled"]; ok {
		p.Enabled = parseBool(v, p.Enabled)
	}
	if v, ok := s.options["auto_update"]; ok {
		p.AutoUpdate = parseBool(v, p.AutoUpdate)
	}
	if v := firstNonEmpty(s.options["update_hour"], s.options["update_interval"], "0"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(v), "h")); err == nil {
			p.UpdateHour = n
		}
	}
	if v := firstNonEmpty(s.options["item_count"], s.options["node_count"]); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			p.ItemCount = n
		}
	}
	return p
}

func LoadRuleDomainFiles(cfg *Config) error {
	if cfg == nil {
		return nil
	}
	for i := range cfg.Rules {
		var domains []string
		for _, path := range cfg.Rules[i].DomainFiles {
			values, err := loadDomainFile(path)
			if err != nil {
				return err
			}
			domains = append(domains, values...)
		}
		for _, providerName := range appendList(cfg.Rules[i].DomainProviders, cfg.Rules[i].Providers) {
			provider, ok := cfg.ProviderByName(providerName)
			if !ok || !provider.Enabled || provider.Type != "domain" {
				continue
			}
			values, err := loadDomainFile(provider.CachePath())
			if err != nil {
				if os.IsNotExist(err) {
					restored, restoreErr := provider.RestoreDefaultCache()
					if restoreErr != nil {
						cfg.Warnings = append(cfg.Warnings, fmt.Sprintf("provider %q cache restore failed: %v", provider.Name, restoreErr))
						continue
					}
					if restored {
						values, err = loadDomainFile(provider.CachePath())
					}
					if os.IsNotExist(err) {
						cfg.Warnings = append(cfg.Warnings, fmt.Sprintf("provider %q cache %q is missing; skipping provider until netod providers update %s", provider.Name, provider.CachePath(), provider.Name))
						continue
					}
				}
			}
			if err != nil {
				return fmt.Errorf("provider %q: %w", provider.Name, err)
			}
			_ = provider.MirrorDefaultCache()
			domains = append(domains, values...)
		}
		if len(domains) > 0 {
			cfg.Rules[i].DomainEquals = cleanDomainList(appendList(cfg.Rules[i].DomainEquals, domains))
		}
	}
	return nil
}

func loadDomainFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []string
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}
		line = strings.TrimRight(strings.ToLower(line), ".")
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%s:%d: %w", path, lineNo, err)
	}
	return cleanDomainList(out), nil
}

func parseOutbound(s section) (Outbound, []string) {
	outbound := Outbound{
		Enabled:              true,
		Tag:                  strings.TrimSpace(firstNonEmpty(s.options["tag"], s.name)),
		Label:                strings.TrimSpace(firstNonEmpty(s.options["label"], s.options["name"], s.options["tag"], s.name)),
		Type:                 strings.TrimSpace(firstNonEmpty(s.options["type"], "vless")),
		Server:               strings.TrimSpace(firstNonEmpty(s.options["server"], s.options["address"])),
		UUID:                 strings.TrimSpace(s.options["uuid"]),
		Flow:                 strings.TrimSpace(firstNonEmpty(s.options["flow"], s.options["vless_flow"])),
		ServerName:           strings.TrimSpace(firstNonEmpty(s.options["server_name"], s.options["tls_sni"])),
		RealityPublicKey:     strings.TrimSpace(firstNonEmpty(s.options["reality_public_key"], s.options["tls_reality_public_key"])),
		RealityShortID:       strings.TrimSpace(firstNonEmpty(s.options["reality_short_id"], s.options["tls_reality_short_id"])),
		TLSMinVersion:        strings.TrimSpace(s.options["tls_min_version"]),
		TLSMaxVersion:        strings.TrimSpace(s.options["tls_max_version"]),
		ECHConfigPath:        strings.TrimSpace(firstNonEmpty(s.options["ech_config_path"], s.options["tls_ech_config_path"])),
		UTLSFingerprint:      strings.TrimSpace(firstNonEmpty(s.options["utls_fingerprint"], s.options["tls_utls"])),
		Transport:            strings.TrimSpace(s.options["transport"]),
		PacketEncoding:       strings.TrimSpace(s.options["packet_encoding"]),
		GRPCServiceName:      strings.TrimSpace(firstNonEmpty(s.options["grpc_service_name"], s.options["grpc_servicename"])),
		HTTPUpgradeHost:      strings.TrimSpace(s.options["httpupgrade_host"]),
		HTTPPath:             strings.TrimSpace(s.options["http_path"]),
		HTTPMethod:           strings.TrimSpace(s.options["http_method"]),
		WSHost:               strings.TrimSpace(s.options["ws_host"]),
		WSPath:               strings.TrimSpace(s.options["ws_path"]),
		WSEarlyDataHeader:    strings.TrimSpace(s.options["websocket_early_data_header"]),
		Password:             strings.TrimSpace(s.options["password"]),
		Method:               strings.TrimSpace(firstNonEmpty(s.options["method"], s.options["shadowsocks_encrypt_method"])),
		ALPN:                 cleanList(appendList(appendList(s.lists["alpn"], s.lists["tls_alpn"]), splitListOption(firstNonEmpty(s.options["alpn"], s.options["tls_alpn"])))),
		TLSCipherSuites:      cleanList(appendList(s.lists["tls_cipher_suites"], splitListOption(s.options["tls_cipher_suites"]))),
		ECHConfig:            cleanList(appendList(appendList(s.lists["ech_config"], s.lists["tls_ech_config"]), splitListOption(firstNonEmpty(s.options["ech_config"], s.options["tls_ech_config"])))),
		HTTPHost:             cleanList(appendList(s.lists["http_host"], splitListOption(s.options["http_host"]))),
		HysteriaObfsType:     strings.TrimSpace(s.options["hysteria_obfs_type"]),
		HysteriaObfsPassword: strings.TrimSpace(s.options["hysteria_obfs_password"]),
	}
	if v := s.options["port"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			outbound.Port = n
		}
	}
	if v, ok := s.options["tls"]; ok {
		outbound.TLS = parseBool(v, outbound.TLS)
	}
	if v, ok := s.options["reality"]; ok {
		outbound.Reality = parseBool(v, outbound.Reality)
	} else if v, ok := s.options["tls_reality"]; ok {
		outbound.Reality = parseBool(v, outbound.Reality)
	}
	if v, ok := s.options["insecure"]; ok {
		outbound.Insecure = parseBool(v, outbound.Insecure)
	} else if v, ok := s.options["tls_insecure"]; ok {
		outbound.Insecure = parseBool(v, outbound.Insecure)
	}
	if v, ok := s.options["ech"]; ok {
		outbound.ECH = parseBool(v, outbound.ECH)
	} else if v, ok := s.options["tls_ech"]; ok {
		outbound.ECH = parseBool(v, outbound.ECH)
	}
	if v := s.options["websocket_early_data"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			outbound.WSEarlyData = n
		}
	}
	if v := s.options["hysteria_up_mbps"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			outbound.HysteriaUpMbps = n
		}
	}
	if v := s.options["hysteria_down_mbps"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			outbound.HysteriaDownMbps = n
		}
	}

	var warnings []string
	if outbound.Tag == "proxy_default" {
		outbound.Enabled = false
		warnings = append(warnings, "outbound 'proxy_default' is deprecated and ignored; create a custom VLESS, Hysteria2, Shadowsocks, or Trojan outbound with its own tag")
	}
	if outbound.Enabled && (outbound.TLS || outbound.Reality) && outbound.ServerName == "" {
		warnings = append(warnings, fmt.Sprintf("outbound %q: server_name is recommended when tls=1 or reality=1", outbound.Tag))
	}
	return outbound, warnings
}

func parseSubscription(s section) Subscription {
	sub := Subscription{
		Name:           strings.TrimSpace(s.name),
		Label:          strings.TrimSpace(firstNonEmpty(s.options["label"], s.options["name"], s.name)),
		Enabled:        true,
		URL:            strings.TrimSpace(s.options["url"]),
		AutoUpdate:     false,
		UpdateInterval: strings.TrimSpace(s.options["update_interval"]),
		UpdateVia:      strings.TrimSpace(firstNonEmpty(s.options["update_via"], "direct")),
		UpdateOutbound: strings.TrimSpace(s.options["update_outbound"]),
		LastUpdate:     strings.TrimSpace(s.options["last_update"]),
	}
	if v, ok := s.options["enabled"]; ok {
		sub.Enabled = parseBool(v, sub.Enabled)
	}
	if v, ok := s.options["auto_update"]; ok {
		sub.AutoUpdate = parseBool(v, sub.AutoUpdate)
	}
	if v := firstNonEmpty(s.options["update_hour"], s.options["update_interval"], "0"); v != "" {
		if n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(v), "h")); err == nil {
			sub.UpdateHour = n
		}
	}
	if v := s.options["node_count"]; v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			sub.NodeCount = n
		}
	}
	return sub
}

func reservedCustomOutboundTag(tag string) bool {
	switch tag {
	case BuiltinDirectOutbound, BuiltinBlockedOutbound, "block", "proxy_default":
		return true
	default:
		return false
	}
}

func requireOutboundFields(outbound Outbound, fields ...string) error {
	for _, field := range fields {
		switch field {
		case "server":
			if strings.TrimSpace(outbound.Server) == "" {
				return fmt.Errorf("outbound %q server is required for type %s", outbound.Tag, outbound.Type)
			}
		case "port":
			if outbound.Port <= 0 || outbound.Port > 65535 {
				return fmt.Errorf("outbound %q port is required for type %s", outbound.Tag, outbound.Type)
			}
		case "uuid":
			if strings.TrimSpace(outbound.UUID) == "" {
				return fmt.Errorf("outbound %q uuid is required for type %s", outbound.Tag, outbound.Type)
			}
		case "password":
			if strings.TrimSpace(outbound.Password) == "" {
				return fmt.Errorf("outbound %q password is required for type %s", outbound.Tag, outbound.Type)
			}
		case "method":
			if strings.TrimSpace(outbound.Method) == "" {
				return fmt.Errorf("outbound %q method is required for type %s", outbound.Tag, outbound.Type)
			}
		}
	}
	return nil
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

func safeProviderName(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

func splitListOption(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
}

func normalizeDNSProtocol(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "", "udp":
		return "udp"
	case "tcp":
		return "tcp"
	case "tls", "dot":
		return "tls"
	case "https", "doh":
		return "https"
	default:
		return strings.ToLower(strings.TrimSpace(protocol))
	}
}

func presetDNSUpstream(preset string, protocol string) (host string, tlsName string, path string) {
	switch strings.TrimSpace(preset) {
	case "google":
		switch normalizeDNSProtocol(protocol) {
		case "tls", "https":
			return "dns.google", "dns.google", "/dns-query"
		}
		return "8.8.8.8", "dns.google", "/dns-query"
	default:
		return "1.1.1.1", "cloudflare-dns.com", "/dns-query"
	}
}

func defaultDNSPort(protocol string) int {
	switch normalizeDNSProtocol(protocol) {
	case "tls":
		return 853
	case "https":
		return 443
	default:
		return 53
	}
}

func syncLegacyDNSFields(m *Main) {
	upstream := m.DNSUpstream()
	if upstream.Host == "" {
		return
	}
	m.RealDNSUpstream = upstream.Address()
	m.DNSUpstreamProtocol = upstream.Protocol
	m.DNSUpstreamHost = upstream.Host
	m.DNSUpstreamPort = upstream.Port
	m.DNSUpstreamTLSName = upstream.TLSName
	m.DNSUpstreamPath = upstream.Path
	if strings.TrimSpace(m.DNSUpstreamPreset) == "" {
		m.DNSUpstreamPreset = "custom"
	}
	if strings.TrimSpace(m.RealDNSTransport) == "" {
		m.RealDNSTransport = upstream.Protocol
	}
	if strings.TrimSpace(m.RealDNSServer) == "" {
		m.RealDNSServer = upstream.Host
	}
	if m.RealDNSServerPort == 0 {
		m.RealDNSServerPort = upstream.Port
	}
	if strings.TrimSpace(m.RealDNSPath) == "" && upstream.Protocol == "https" {
		m.RealDNSPath = upstream.Path
	}
}

func splitHostPortValue(value string) (string, int, bool) {
	host, portValue, err := net.SplitHostPort(strings.TrimSpace(value))
	if err != nil {
		return "", 0, false
	}
	port, err := strconv.Atoi(portValue)
	if err != nil {
		return "", 0, false
	}
	return host, port, true
}

func splitOptionalHostPort(value string) (string, int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0, false
	}
	host, port, ok := splitHostPortValue(value)
	if ok {
		return host, port, true
	}
	if strings.Count(value, ":") == 1 {
		parts := strings.Split(value, ":")
		if p, err := strconv.Atoi(parts[1]); err == nil && p > 0 {
			return parts[0], p, true
		}
	}
	return value, 0, false
}

func sameHost(a string, b string) bool {
	a = strings.Trim(strings.ToLower(strings.TrimSpace(a)), "[]")
	b = strings.Trim(strings.ToLower(strings.TrimSpace(b)), "[]")
	if a == b {
		return true
	}
	aIP := net.ParseIP(a)
	bIP := net.ParseIP(b)
	return aIP != nil && bIP != nil && aIP.Equal(bIP)
}

func routerLANIP(lanSubnets []string, ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	for _, cidr := range lanSubnets {
		base, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		base4 := base.To4()
		if base4 == nil {
			continue
		}
		router := append(net.IP(nil), base4...)
		router[3]++
		if network.Contains(router) && router.Equal(ip4) {
			return true
		}
	}
	return false
}

func ruleHasDomainSelectors(r Rule, providers map[string]Provider) bool {
	if len(r.DomainEquals)+len(r.DomainContains)+len(r.DomainStartsWith)+len(r.DomainEndsWith)+len(r.DomainFiles)+len(r.DomainProviders) > 0 {
		return true
	}
	for _, name := range r.Providers {
		if provider, ok := providers[strings.TrimSpace(name)]; ok && provider.Type == "domain" {
			return true
		}
	}
	return false
}

func ruleHasIPSelectors(r Rule, providers map[string]Provider) bool {
	if len(r.IPCIDRs)+len(r.Files)+len(r.IPProviders) > 0 {
		return true
	}
	for _, name := range r.Providers {
		if provider, ok := providers[strings.TrimSpace(name)]; ok && provider.Type == "ip" {
			return true
		}
	}
	return false
}

func ruleHasPortSelectors(r Rule) bool {
	return len(r.SrcPorts)+len(r.DstPorts) > 0
}

func parseRulePortRange(value string) (int, int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, 0, fmt.Errorf("empty port")
	}
	parts := strings.Split(value, "-")
	if len(parts) > 2 {
		return 0, 0, fmt.Errorf("expected port or start-end")
	}
	start, err := parseRulePort(parts[0])
	if err != nil {
		return 0, 0, err
	}
	end := start
	if len(parts) == 2 {
		end, err = parseRulePort(parts[1])
		if err != nil {
			return 0, 0, err
		}
		if start > end {
			return 0, 0, fmt.Errorf("range start must be <= end")
		}
	}
	return start, end, nil
}

func parseRulePort(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("empty port")
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if n < 1 || n > 65535 {
		return 0, fmt.Errorf("port must be 1..65535")
	}
	return n, nil
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
	escaped := false
	for i, r := range s {
		if escaped {
			escaped = false
			continue
		}
		switch {
		case quote != 0:
			if r == '\\' {
				escaped = true
				continue
			}
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
	escaped := false
	inField := false

	flush := func() {
		if inField {
			fields = append(fields, b.String())
			b.Reset()
			inField = false
		}
	}

	for _, r := range s {
		if escaped {
			b.WriteRune(r)
			inField = true
			escaped = false
			continue
		}
		switch {
		case quote != 0:
			if r == '\\' {
				escaped = true
				inField = true
				continue
			}
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

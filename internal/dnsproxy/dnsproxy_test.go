package dnsproxy

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/elllkere/neto/internal/config"
)

func testQuery(qtype uint16) []byte {
	msg := []byte{
		0x12, 0x34, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x07, 'y', 'o', 'u', 't', 'u', 'b', 'e',
		0x03, 'c', 'o', 'm',
		0x00, 0x00, 0x01, 0x00, 0x01,
	}
	binary.BigEndian.PutUint16(msg[len(msg)-4:len(msg)-2], qtype)
	return msg
}

func TestQueryName(t *testing.T) {
	msg := testQuery(qTypeA)
	got, ok := QueryName(msg)
	if !ok || got != "youtube.com" {
		t.Fatalf("got %q, %t", got, ok)
	}
}

func TestParseQuery(t *testing.T) {
	query, ok := ParseQuery(testQuery(qTypeAAAA))
	if !ok {
		t.Fatal("query did not parse")
	}
	if query.Name != "youtube.com" || query.Type != qTypeAAAA || query.QuestionEnd != len(testQuery(qTypeAAAA)) {
		t.Fatalf("unexpected query: %+v", query)
	}
}

func TestMatchesFakeIP(t *testing.T) {
	p := Proxy{Rules: []config.Rule{
		fakeRule("youtube.com"),
		fakeRule("googlevideo.com"),
	}}
	for _, name := range []string{"youtube.com", "www.youtube.com", "r1.googlevideo.com"} {
		decision := p.domainDecision(name, "192.168.8.10")
		if decision.Action != "proxy" || decision.DNSMode != "fakeip" {
			t.Fatalf("expected %s to match", name)
		}
	}
	if decision := p.domainDecision("notyoutube.com", "192.168.8.10"); decision.Action == "proxy" && decision.DNSMode == "fakeip" {
		t.Fatal("unexpected root/subdomain match")
	}
}

func TestFakeIPAAAAGetsLocalNODATA(t *testing.T) {
	p := Proxy{
		Rules:               []config.Rule{fakeRule("youtube.com")},
		RoutingMode:         "custom",
		FilterAAAAForFakeIP: true,
	}
	resp, ok := p.localResponse(testQuery(qTypeAAAA), "192.168.8.10")
	if !ok {
		t.Fatal("expected local AAAA response")
	}
	if len(resp) != len(testQuery(qTypeAAAA)) {
		t.Fatalf("unexpected response length %d", len(resp))
	}
	if resp[2]&0x80 == 0 {
		t.Fatal("response bit is not set")
	}
	if rcode := resp[3] & 0x0f; rcode != 0 {
		t.Fatalf("rcode=%d, want NOERROR", rcode)
	}
	if answers := binary.BigEndian.Uint16(resp[6:8]); answers != 0 {
		t.Fatalf("answers=%d, want 0", answers)
	}
}

func TestFakeIPAStillForwards(t *testing.T) {
	p := Proxy{
		Rules:               []config.Rule{fakeRule("youtube.com")},
		RoutingMode:         "custom",
		FilterAAAAForFakeIP: true,
	}
	if _, ok := p.localResponse(testQuery(qTypeA), "192.168.8.10"); ok {
		t.Fatal("A query must be forwarded to sing-box FakeIP DNS")
	}
}

func TestDNSPolicyCustomDefaultFakeIP(t *testing.T) {
	p := Proxy{
		Rules:        []config.Rule{fakeRule("youtube.com")},
		RoutingMode:  "custom",
		FakeUpstream: "127.0.0.1:15353",
		RealUpstream: "1.1.1.1:53",
	}
	if got := p.upstreamFor(testQuery(qTypeA), "192.168.8.10"); got != p.FakeUpstream {
		t.Fatalf("got %s, want fake upstream", got)
	}
	if got := p.upstreamFor(testQueryName(qTypeA, "example.org"), "192.168.8.10"); got != p.RealUpstream {
		t.Fatalf("got %s, want real upstream", got)
	}
}

func TestDNSPolicyCustomDirectNoFakeIP(t *testing.T) {
	p := Proxy{
		Rules:          []config.Rule{fakeRule("youtube.com")},
		RoutingMode:    "custom",
		FakeUpstream:   "127.0.0.1:15353",
		RealUpstream:   "1.1.1.1:53",
		ClientPolicies: map[string]string{"192.168.8.50": "direct"},
	}
	if got := p.upstreamFor(testQuery(qTypeA), "192.168.8.50"); got != p.RealUpstream {
		t.Fatalf("got %s, want real upstream for direct client", got)
	}
	if _, ok := p.localResponse(testQuery(qTypeAAAA), "192.168.8.50"); ok {
		t.Fatal("direct client AAAA must not receive fakeip local response")
	}
}

func TestDNSPolicyGlobalDefaultReal(t *testing.T) {
	p := Proxy{
		Rules:        []config.Rule{fakeRule("youtube.com")},
		RoutingMode:  "global",
		FakeUpstream: "127.0.0.1:15353",
		RealUpstream: "1.1.1.1:53",
	}
	if got := p.upstreamFor(testQuery(qTypeA), "192.168.8.10"); got != p.RealUpstream {
		t.Fatalf("got %s, want real upstream in global default", got)
	}
}

func TestDNSPolicyGlobalProxyCanUseFakeIP(t *testing.T) {
	p := Proxy{
		Rules:          []config.Rule{fakeRule("youtube.com")},
		RoutingMode:    "global",
		FakeUpstream:   "127.0.0.1:15353",
		RealUpstream:   "1.1.1.1:53",
		ClientPolicies: map[string]string{"192.168.8.100": "proxy"},
	}
	if got := p.upstreamFor(testQuery(qTypeA), "192.168.8.100"); got != p.FakeUpstream {
		t.Fatalf("got %s, want fake upstream for proxy client matched domain", got)
	}
}

func TestDNSPolicyBlockReturnsNXDOMAIN(t *testing.T) {
	p := Proxy{
		Rules: []config.Rule{{
			Name:         "blocked",
			Enabled:      true,
			Action:       "block",
			DNSMode:      "real_ip",
			DomainEquals: []string{"blocked.example"},
		}},
		RoutingMode: "custom",
	}
	resp, ok := p.localResponse(testQueryName(qTypeA, "blocked.example"), "192.168.8.10")
	if !ok {
		t.Fatal("expected local block response")
	}
	if rcode := resp[3] & 0x0f; rcode != 3 {
		t.Fatalf("rcode=%d, want NXDOMAIN", rcode)
	}
}

func fakeRule(suffix string) config.Rule {
	return config.Rule{
		Name:           suffix,
		Enabled:        true,
		Action:         "proxy",
		Outbound:       "proxy_default",
		DNSMode:        "fakeip",
		DomainEquals:   []string{suffix},
		DomainEndsWith: []string{"." + suffix},
	}
}

func testQueryName(qtype uint16, name string) []byte {
	msg := []byte{0x12, 0x34, 0x01, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}
	for _, label := range strings.Split(name, ".") {
		msg = append(msg, byte(len(label)))
		msg = append(msg, []byte(label)...)
	}
	msg = append(msg, 0, 0, 0, 0, 1)
	binary.BigEndian.PutUint16(msg[len(msg)-4:len(msg)-2], qtype)
	return msg
}

func TestClientSubnetIPv4(t *testing.T) {
	msg := testQuery(qTypeA)
	binary.BigEndian.PutUint16(msg[10:12], 1)
	opt := []byte{
		0x00,       // root name
		0x00, 0x29, // OPT
		0x04, 0xd0, // UDP payload size
		0x00, 0x00, 0x00, 0x00, // extended rcode/version/flags
		0x00, 0x0c, // rdlen
		0x00, 0x08, // ECS option
		0x00, 0x08, // option len
		0x00, 0x01, // family IPv4
		0x20, // source prefix 32
		0x00, // scope prefix
		192, 168, 8, 50,
	}
	msg = append(msg, opt...)
	if got := clientSubnetIPv4(msg); got != "192.168.8.50" {
		t.Fatalf("got %q, want 192.168.8.50", got)
	}
	p := Proxy{}
	if got := p.clientIP(msg, "127.0.0.1"); got != "192.168.8.50" {
		t.Fatalf("got client IP %q", got)
	}
}

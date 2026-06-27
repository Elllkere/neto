package provider

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/elllkere/neto/internal/config"
	"github.com/elllkere/neto/internal/policy"
)

func TestLoadRuleCIDRsCombinesInlineAndFileCIDRs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cidrs.txt")
	if err := os.WriteFile(path, []byte("8.8.8.0/24\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Defaults()
	providerPath := filepath.Join(dir, "provider.txt")
	if err := os.WriteFile(providerPath, []byte("9.9.9.0/24\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg.Rules = []config.Rule{{
		Name:        "ip",
		Enabled:     true,
		Action:      "proxy",
		DNSMode:     "real_ip",
		IPProviders: []string{"remote_ips"},
		IPCIDRs: []string{
			"1.1.1.1",
		},
		Files: []string{
			path,
		},
	}}
	cfg.Providers = []config.Provider{{
		Name:      "remote_ips",
		Enabled:   true,
		Type:      "ip",
		LocalPath: providerPath,
		URL:       "https://example.com/ip.txt",
	}}

	got, err := LoadRuleCIDRs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	values := policy.CIDRStrings(got[0])
	want := []string{"1.1.1.1/32", "8.8.8.0/24", "9.9.9.0/24"}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("got %v, want %v", values, want)
	}
}

func TestNormalizeDownloadedProviderLists(t *testing.T) {
	domains, err := NormalizeDownloadedList(config.Provider{Name: "domains", Type: "domain"}, []byte("Example.COM.\nexample.org # comment\n\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(domains, []string{"example.com", "example.org"}) {
		t.Fatalf("unexpected domains: %v", domains)
	}

	cidrs, err := NormalizeDownloadedList(config.Provider{Name: "ips", Type: "ip"}, []byte("1.1.1.1\n8.8.8.0/24\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cidrs, []string{"1.1.1.1/32", "8.8.8.0/24"}) {
		t.Fatalf("unexpected cidrs: %v", cidrs)
	}
}

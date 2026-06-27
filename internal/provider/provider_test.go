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
	cfg.Rules = []config.Rule{{
		Name:    "ip",
		Enabled: true,
		Action:  "proxy",
		DNSMode: "real_ip",
		IPCIDRs: []string{
			"1.1.1.1",
		},
		Files: []string{
			path,
		},
	}}

	got, err := LoadRuleCIDRs(cfg)
	if err != nil {
		t.Fatal(err)
	}
	values := policy.CIDRStrings(got[0])
	want := []string{"1.1.1.1/32", "8.8.8.0/24"}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("got %v, want %v", values, want)
	}
}

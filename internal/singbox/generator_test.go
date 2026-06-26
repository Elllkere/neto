package singbox

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/elllkere/neto/internal/config"
)

func TestGenerateUsesModernFakeIPServer(t *testing.T) {
	cfg := config.Defaults()
	out, err := Generate(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatal(err)
	}
	raw := string(out)
	if !strings.Contains(raw, `"type": "fakeip"`) || !strings.Contains(raw, `"tag": "fakeip"`) {
		t.Fatalf("fakeip server missing:\n%s", raw)
	}
	if strings.Contains(raw, `"fake-ip"`) || strings.Contains(raw, `"fake_ip"`) {
		t.Fatalf("legacy fake-ip config detected:\n%s", raw)
	}
	if !strings.Contains(raw, `"listen_port": 15353`) || !strings.Contains(raw, `"listen_port": 16001`) {
		t.Fatalf("expected DNS and TProxy listeners:\n%s", raw)
	}
	if !strings.Contains(raw, `"default_domain_resolver": "local"`) {
		t.Fatalf("expected route.default_domain_resolver:\n%s", raw)
	}
}

func TestCompareVersion(t *testing.T) {
	if compareVersion("1.11.9", MinimumVersion) >= 0 {
		t.Fatal("1.11.9 should be unsupported")
	}
	if compareVersion("1.12.0", MinimumVersion) < 0 {
		t.Fatal("1.12.0 should be supported")
	}
	if compareVersion("1.13.12", MinimumVersion) < 0 {
		t.Fatal("1.13.12 should be supported")
	}
}

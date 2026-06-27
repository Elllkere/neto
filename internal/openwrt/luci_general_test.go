package openwrt

import (
	"os"
	"strings"
	"testing"
)

func TestGeneralLuCIFlagsPersistZeroValues(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/general.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, field := range []string{
		"enabled",
		"manage_dnsmasq",
		"filter_aaaa_for_fakeip",
		"fakeip_enabled",
		"resolve_for_subnet_rules",
		"nft_counters",
	} {
		needle := "form.Flag, '" + field + "'"
		start := strings.Index(s, needle)
		if start < 0 {
			t.Fatalf("general.js missing flag %q:\n%s", field, s)
		}
		end := strings.Index(s[start+len(needle):], "\n\t\to = s.option(")
		block := s[start:]
		if end >= 0 {
			block = s[start : start+len(needle)+end]
		}
		for _, want := range []string{
			"o.enabled = '1'",
			"o.disabled = '0'",
			"o.rmempty = false",
		} {
			if !strings.Contains(block, want) {
				t.Fatalf("flag %q must persist 0/1 and not delete option; missing %q:\n%s", field, want, block)
			}
		}
	}
}

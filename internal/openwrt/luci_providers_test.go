package openwrt

import (
	"os"
	"strings"
	"testing"
)

func TestProvidersLuCIUsesProviderSections(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/providers.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		"form.GridSection, 'provider'",
		"form.Value, 'name'",
		"form.DynamicList, 'file'",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("providers.js missing %q:\n%s", want, s)
		}
	}
	for _, forbidden := range []string{
		"form.GridSection, 'rule'",
		"form.Value, 'priority'",
		"form.ListValue, 'action'",
		"form.ListValue, 'dns_mode'",
		"form.ListValue, 'outbound'",
		"sortable = true",
	} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("providers.js must not contain policy field %q:\n%s", forbidden, s)
		}
	}
}

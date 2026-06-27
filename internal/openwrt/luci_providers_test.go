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
		"form.Flag, 'enabled'",
		"form.Value, 'label'",
		"form.ListValue, 'type'",
		"form.Value, 'url'",
		"form.Flag, 'auto_update'",
		"form.ListValue, 'update_hour'",
		"form.ListValue, 'update_via'",
		"form.ListValue, 'update_outbound'",
		"form.DummyValue, 'item_count'",
		"form.Button, '_update'",
		"fs.exec('/usr/bin/netod', [ 'providers', 'update', section_id ])",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("providers.js missing %q:\n%s", want, s)
		}
	}
	for _, forbidden := range []string{
		"form.GridSection, 'rule'",
		"form.DynamicList, 'file'",
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

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
		"o.value('direct', 'direct')",
		"o.value('proxy', 'proxy')",
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

func TestProvidersLuCITableEditsOnlyFlags(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/providers.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, needle := range []string{
		"form.ListValue, 'type'",
		"form.Value, 'url'",
		"form.ListValue, 'update_via'",
	} {
		start := strings.Index(s, needle)
		if start < 0 {
			t.Fatalf("providers.js missing field %q:\n%s", needle, s)
		}
		end := strings.Index(s[start+len(needle):], "\n\t\to = s.option(")
		block := s[start:]
		if end >= 0 {
			block = s[start : start+len(needle)+end]
		}
		if strings.Contains(block, "o.editable = true") {
			t.Fatalf("provider table field %q should be read-only text:\n%s", needle, block)
		}
		if strings.Contains(block, "o.modalonly = true") {
			t.Fatalf("provider table field %q should remain visible:\n%s", needle, block)
		}
	}
	for _, needle := range []string{
		"form.Flag, 'enabled'",
		"form.Flag, 'auto_update'",
	} {
		start := strings.Index(s, needle)
		if start < 0 {
			t.Fatalf("providers.js missing flag %q:\n%s", needle, s)
		}
		end := strings.Index(s[start+len(needle):], "\n\t\to = s.option(")
		block := s[start:]
		if end >= 0 {
			block = s[start : start+len(needle)+end]
		}
		if !strings.Contains(block, "o.editable = true") {
			t.Fatalf("provider flag %q should remain editable in table:\n%s", needle, block)
		}
	}
}

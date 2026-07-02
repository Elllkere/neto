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
		"form.ListValue, 'source'",
		"form.Value, 'url'",
		"form.Value, 'script_path'",
		"NETO_PROVIDER_OUTPUT",
		"form.Flag, 'auto_update'",
		"form.ListValue, 'update_hour'",
		"form.ListValue, 'update_minute'",
		"form.ListValue, 'update_via'",
		"o.value('direct', 'direct')",
		"o.value('proxy', 'proxy')",
		"form.ListValue, 'update_outbound'",
		"form.DummyValue, 'item_count'",
		"form.DummyValue, 'last_update'",
		"validateProviderReferences()",
		"referencedProviders(section_id)",
		"handleSaveCommit: function()",
		"return uci.commit('neto')",
		"domain_provider",
		"ip_provider",
		"Rule \"%s\" references missing or disabled provider \"%s\"",
		"form.Button, '_update'",
		"function(ev, section_id)",
		"NETO_PROVIDER_PROXY",
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
		"form.Value, 'description'",
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

func TestProvidersLuCIShowsUpdatedInTable(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/providers.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	needle := "form.DummyValue, 'last_update'"
	start := strings.Index(s, needle)
	if start < 0 {
		t.Fatalf("providers.js missing field %q:\n%s", needle, s)
	}
	end := strings.Index(s[start+len(needle):], "\n\t\to = s.option(")
	block := s[start:]
	if end >= 0 {
		block = s[start : start+len(needle)+end]
	}
	if strings.Contains(block, "o.modalonly = true") {
		t.Fatalf("provider updated field should remain visible in table:\n%s", block)
	}
}

func TestProvidersLuCIUpdateButtonIsModalOnly(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/providers.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	needle := "form.Button, '_update'"
	start := strings.Index(s, needle)
	if start < 0 {
		t.Fatalf("providers.js missing field %q:\n%s", needle, s)
	}
	end := strings.Index(s[start+len(needle):], "\n\t\to = s.option(")
	block := s[start:]
	if end >= 0 {
		block = s[start : start+len(needle)+end]
	}
	for _, want := range []string{
		"o.inputtitle = _('Update')",
		"return true;",
		"o.modalonly = true",
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("provider update button missing %q:\n%s", want, block)
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
		"form.ListValue, 'source'",
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
		for _, forbidden := range []string{
			"plain text provider list",
			"custom filtering",
			"NETO_PROVIDER_PROXY",
		} {
			if strings.Contains(block, forbidden) {
				t.Fatalf("provider table field %q should not contain help text %q:\n%s", needle, forbidden, block)
			}
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

func TestProvidersLuCIURLOnlyAppliesToURLSource(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/providers.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	needle := "form.Value, 'url'"
	start := strings.Index(s, needle)
	if start < 0 {
		t.Fatalf("providers.js missing field %q:\n%s", needle, s)
	}
	end := strings.Index(s[start+len(needle):], "\n\t\to = s.option(")
	block := s[start:]
	if end >= 0 {
		block = s[start : start+len(needle)+end]
	}
	if !strings.Contains(block, "o.depends('source', 'url')") {
		t.Fatalf("provider URL should only be active for source=url:\n%s", block)
	}
}

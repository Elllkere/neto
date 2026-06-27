package openwrt

import (
	"os"
	"strings"
	"testing"
)

func TestRulesLuCINoMatchAllOrLegacyMatchers(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/rules.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, forbidden := range []string{
		"match_all",
		"domain_exact",
		"domain_keyword",
		"domain_prefix",
		"domain_suffix",
		"exclude_domain_exact",
		"exclude_domain_keyword",
		"exclude_domain_prefix",
		"exclude_domain_suffix",
		"Keyword",
		"Prefix",
		"Suffix",
	} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("rules.js must not contain %q:\n%s", forbidden, s)
		}
	}
}

func TestRulesLuCIExplicitEnabledAndPriorityRewrite(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/rules.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		"form.Flag, 'enabled'",
		"o.enabled = '1'",
		"o.disabled = '0'",
		"o.rmempty = false",
		"uci.set('neto', sid, 'enabled', '1')",
		"uci.get('neto', sid, 'action') == 'proxy'",
		"uci.set('neto', sid, 'dns_mode', 'fakeip')",
		"uci.set('neto', sid, 'priority', String(n * 100))",
		"this.map.save(rewriteRuleState)",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("rules.js missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "on_before_save") {
		t.Fatalf("rules.js must not use unsupported on_before_save hook:\n%s", s)
	}
}

func TestRulesLuCIDNSModeHiddenAndForcedToFakeIP(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/rules.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if strings.Contains(s, "s.option(form.ListValue, 'dns_mode'") || strings.Contains(s, "DNS mode") {
		t.Fatalf("rules.js must not expose dns_mode in LuCI:\n%s", s)
	}
	for _, want := range []string{
		"uci.get('neto', sid, 'action') == 'proxy'",
		"uci.set('neto', sid, 'dns_mode', 'fakeip')",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("rules.js missing forced fakeip behavior %q:\n%s", want, s)
		}
	}
}

func TestRulesLuCICompactTableFields(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/rules.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, field := range []string{
		"domain_equals",
		"domain_contains",
		"domain_starts_with",
		"domain_ends_with",
		"exclude_domain_equals",
		"exclude_domain_contains",
		"exclude_domain_starts_with",
		"exclude_domain_ends_with",
		"file",
	} {
		needle := "s.option(form.DynamicList, '" + field + "'"
		if !strings.Contains(s, needle) {
			t.Fatalf("rules.js missing detail field %q:\n%s", field, s)
		}
	}
	if strings.Count(s, "o.modalonly = true") < 9 {
		t.Fatalf("matcher/provider fields should be modal-only:\n%s", s)
	}
	if strings.Contains(s, "form.Value, 'priority'") {
		t.Fatalf("priority must not be user-editable:\n%s", s)
	}
	if strings.Contains(s, "form.GridSection, 'provider'") {
		t.Fatalf("rules.js must not create provider sections:\n%s", s)
	}
}

func TestRulesLuCIOutboundVisibleInTable(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/rules.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	start := strings.Index(s, "form.ListValue, 'outbound'")
	end := strings.Index(s, "form.DynamicList, 'domain_equals'")
	if start < 0 || end < 0 || end <= start {
		t.Fatalf("could not find outbound option block:\n%s", s)
	}
	block := s[start:end]
	if strings.Contains(block, "modalonly") {
		t.Fatalf("outbound must remain visible in rules table:\n%s", block)
	}
	if !strings.Contains(block, "o.editable = true") {
		t.Fatalf("outbound should be editable in rules table:\n%s", block)
	}
	if !strings.Contains(block, "addOutboundChoices(o)") || !strings.Contains(block, "o.default = 'direct'") {
		t.Fatalf("outbound should use dynamic choices and default direct:\n%s", block)
	}
	for _, want := range []string{
		"option.value('direct'",
		"option.value('blocked'",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("rules.js missing builtin outbound choice %q:\n%s", want, s)
		}
	}
	if strings.Contains(block, "proxy_default") {
		t.Fatalf("rule outbound dropdown must not add proxy_default:\n%s", block)
	}
	if strings.Contains(s, "section.enabled") {
		t.Fatalf("rules.js must not filter custom outbounds by removed enabled option:\n%s", s)
	}
	for _, want := range []string{
		"var tag = String(section.tag || sid || section['.name'] || '').trim()",
		"var label = String(section.label || section.name || tag).trim()",
		"option.value(tag, label || tag)",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("rules.js should support pending custom outbounds; missing %q:\n%s", want, s)
		}
	}
}

package openwrt

import (
	"os"
	"strings"
	"testing"
)

func TestOutboundsLuCIExposesNativeTypes(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/outbounds.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		"form.GridSection, 'outbound'",
		"s.addremove = true",
		"s.modaltitle = _('Outbound details')",
		"s.sectiontitle = function(section_id)",
		"s.renderSectionAdd = function()",
		"ui.addValidator(nameEl, 'uciname'",
		"function outboundTagExists(tag)",
		"section.tag || sid || section['.name']",
		"addNamedSectionValidator(el, this, _('This tag is reserved'), true)",
		"return _('Expecting: %s').format(_('unique outbound tag'))",
		"form.Value, 'label'",
		"o.cfgvalue = function(section_id)",
		"o.write = function(section_id, formvalue)",
		"uci.set('neto', section_id, 'label', label || section_id)",
		"uci.set('neto', sid, 'tag', sid)",
		"o.value('vless'",
		"o.value('hysteria2'",
		"o.value('shadowsocks'",
		"o.value('trojan'",
		"o.default = 'vless'",
		"form.Value, 'server', _('Address')",
		"uci.get('neto', section_id, 'server') || uci.get('neto', section_id, 'address')",
		"uci.set('neto', section_id, 'server', String(formvalue || '').trim())",
		"reality_public_key",
		"tls_min_version",
		"tls_max_version",
		"tls_cipher_suites",
		"ech_config",
		"utls_fingerprint",
		"grpc_service_name",
		"websocket_early_data",
		"packet_encoding",
		"hysteria_obfs_type",
		"method",
		"password",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("outbounds.js missing %q:\n%s", want, s)
		}
	}
	for _, forbidden := range []string{
		"o.value('socks'",
		"o.value('socks4'",
		"o.value('socks5'",
		"o.value('mixed'",
		"o.default = 'proxy_default'",
		"uci.set('neto', 'proxy_default', 'outbound')",
		"form.value, 'tag'",
	} {
		if strings.Contains(strings.ToLower(s), forbidden) {
			t.Fatalf("outbounds.js must not expose %q:\n%s", forbidden, s)
		}
	}
	if strings.Count(s, "o.modalonly = true") < 10 {
		t.Fatalf("outbound detail fields should be modal-only:\n%s", s)
	}
	start := strings.Index(s, "form.ListValue, 'type'")
	end := strings.Index(s, "form.Value, 'server', _('Address')")
	if start < 0 || end < 0 || end <= start {
		t.Fatalf("could not find outbound type block:\n%s", s)
	}
	typeBlock := s[start:end]
	if strings.Contains(typeBlock, "o.value('direct'") {
		t.Fatalf("outbound type dropdown must not expose direct:\n%s", typeBlock)
	}
	if strings.Contains(typeBlock, "o.editable = true") {
		t.Fatalf("outbound type should be read-only text in table and editable in modal:\n%s", typeBlock)
	}
}

func TestOutboundsLuCITableOnlySectionNameTypeAddressPort(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/outbounds.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "s.sectiontitle = function(section_id)") {
		t.Fatalf("outbounds table should show name through section title:\n%s", s)
	}
	visible := []string{
		"form.ListValue, 'type'",
		"form.Value, 'server', _('Address')",
		"form.Value, 'port'",
	}
	for _, needle := range visible {
		start := strings.Index(s, needle)
		if start < 0 {
			t.Fatalf("outbounds.js missing visible field %q:\n%s", needle, s)
		}
		end := strings.Index(s[start+len(needle):], "\n\t\to = s.option(")
		block := s[start:]
		if end >= 0 {
			block = s[start : start+len(needle)+end]
		}
		if strings.Contains(block, "o.modalonly = true") {
			t.Fatalf("field %q should be visible in table:\n%s", needle, block)
		}
		if strings.Contains(block, "o.depends(") {
			t.Fatalf("field %q should not depend on the modal type control in the table:\n%s", needle, block)
		}
		if strings.Contains(block, "o.editable = true") {
			t.Fatalf("field %q should be read-only text in the table and editable through the modal:\n%s", needle, block)
		}
	}
	for _, needle := range []string{
		"form.Value, 'label'",
		"form.Value, 'uuid'",
		"form.ListValue, 'flow'",
		"form.Flag, 'tls'",
		"form.Value, 'server_name'",
		"form.Flag, 'reality'",
		"form.Value, 'reality_public_key'",
		"form.Value, 'password'",
		"form.ListValue, 'method'",
	} {
		start := strings.Index(s, needle)
		if start < 0 {
			t.Fatalf("outbounds.js missing detail field %q:\n%s", needle, s)
		}
		end := strings.Index(s[start+len(needle):], "\n\t\to = s.option(")
		block := s[start:]
		if end >= 0 {
			block = s[start : start+len(needle)+end]
		}
		if !strings.Contains(block, "o.modalonly = true") {
			t.Fatalf("field %q should be modal-only:\n%s", needle, block)
		}
	}
}

func TestOutboundsLuCIHomeProxyLikeControlsAndDependencies(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/outbounds.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		"form.ListValue, 'flow'",
		"o.value('xtls-rprx-vision')",
		"form.ListValue, 'tls_min_version'",
		"form.ListValue, 'tls_max_version'",
		"form.DynamicList, 'tls_cipher_suites'",
		"form.Flag, 'insecure', _('Allow insecure')",
		"form.Flag, 'ech', _('Enable ECH')",
		"form.DynamicList, 'ech_config'",
		"form.Value, 'ech_config_path'",
		"form.ListValue, 'utls_fingerprint'",
		"o.value('chrome')",
		"form.Flag, 'reality'",
		"o.depends({ 'type': 'vless', 'tls': '1' })",
		"form.Value, 'reality_public_key'",
		"function dependsReality(option)",
		"option.depends({ 'type': 'vless', 'tls': '1', 'reality': '1' })",
		"form.ListValue, 'transport'",
		"o.value('grpc', _('gRPC'))",
		"o.value('ws', _('WebSocket'))",
		"form.ListValue, 'packet_encoding'",
		"form.ListValue, 'method', _('Encrypt method')",
		"o.value('2022-blake3-aes-128-gcm')",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("outbounds.js missing homeproxy-like control/dependency %q:\n%s", want, s)
		}
	}
}

func TestOutboundsLuCIMenuEntry(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/usr/share/luci/menu.d/luci-app-neto.json")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, `"admin/services/neto/outbounds"`) || !strings.Contains(s, `"path": "neto/outbounds"`) {
		t.Fatalf("menu missing outbounds page:\n%s", s)
	}
}

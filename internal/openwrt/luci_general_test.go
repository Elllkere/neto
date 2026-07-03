package openwrt

import (
	"os"
	"strings"
	"testing"
)

func TestGeneralLuCIShowsStatusControlsAndOnlyCoreSettings(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/general.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		"'require neto.i18n as netoI18n'",
		"commandResult('/etc/init.d/neto', [ 'status' ])",
		"commandResult('/etc/init.d/neto', [ 'enabled' ])",
		"commandResult('/bin/pidof', [ 'netod' ])",
		"commandResult('/bin/pidof', [ 'sing-box' ])",
		"commandResult('/usr/bin/netod', [ 'version' ])",
		"commandResult(singboxBin, [ 'version' ])",
		"form.DummyValue, '_neto_status'",
		"form.DummyValue, '_singbox_status'",
		"form.DummyValue, '_netod_version'",
		"form.DummyValue, '_singbox_version'",
		"form.Button, '_service'",
		"form.Button, '_autostart'",
		"fs.exec('/etc/init.d/neto', [ action ])",
		"fs.exec('/sbin/uci', [ 'set', 'neto.main.enabled=1' ])",
		"form.ListValue, 'dns_upstream_preset'",
		"o.value('cloudflare', _('Cloudflare'))",
		"o.value('google', _('Google'))",
		"o.value('custom', _('Custom'))",
		"form.ListValue, 'real_dns_mode'",
		"o.value('direct', 'direct')",
		"o.value('proxy', 'proxy')",
		"form.ListValue, 'real_dns_outbound'",
		"addProxyOutboundChoices(o)",
		"o.depends('real_dns_mode', 'proxy')",
		"form.ListValue, 'real_dns_transport'",
		"form.Value, 'real_dns_server'",
		"form.Value, 'real_dns_server_name'",
		"form.Value, '_real_dns_doh'",
		"splitDoHValue(formvalue",
		"port = defaultDNSPort(protocol)",
		"uci.set('neto', 'main', 'real_dns_server', host + ':' + port)",
		"uci.set('neto', 'main', 'real_dns_transport', protocol)",
		"uci.set('neto', 'main', 'real_dns_upstream', host + ':' + port)",
		"form.ListValue, 'routing_mode'",
		"o.value('simple', _('Simple'))",
		"form.ListValue, 'default_outbound'",
		"o.value('direct', 'direct')",
		"form.ListValue, 'simple_action'",
		"form.ListValue, 'simple_outbound'",
		"addSimpleProviderList(s, 'simple_domain_provider'",
		"addSimpleProviderList(s, 'simple_ip_provider'",
		"form.DynamicList, 'simple_domain_equals'",
		"form.DynamicList, 'simple_domain_ends_with'",
		"form.DynamicList, 'simple_ip_cidr'",
		"normalizeSimpleRuleState()",
		"if (routingMode != 'simple')",
		"o.retain = true",
		"form.ListValue, 'language'",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("general.js missing %q:\n%s", want, s)
		}
	}
	for _, forbidden := range []string{
		"form.Flag, 'enabled'",
		"form.Flag, 'fakeip_enabled'",
		"form.Value, 'dns_listen'",
		"form.Value, 'fakeip_range'",
		"form.Flag, 'filter_aaaa_for_fakeip'",
		"form.DynamicList, 'lan_subnet'",
		"form.Value, 'singbox_bin'",
		"form.Value, 'tproxy_port'",
	} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("general.js should not expose %q:\n%s", forbidden, s)
		}
	}
}

func TestAdvancedLuCIContainsMovedSettingsButNoFakeIPToggle(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/advanced.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		"form.NamedSection, 'main', 'main', _('Advanced')",
		"form.Flag, 'manage_dnsmasq'",
		"form.Value, 'dns_listen'",
		"form.Flag, 'filter_aaaa_for_fakeip'",
		"form.DynamicList, 'lan_subnet'",
		"form.DynamicList, 'lan_iface'",
		"form.Value, 'singbox_bin'",
		"form.Value, 'singbox_dns_fakeip'",
		"form.Value, 'singbox_dns_real_direct'",
		"form.Value, 'singbox_dns_real_proxy'",
		"form.Value, 'tproxy_port'",
		"form.Value, 'mark'",
		"form.Value, 'table'",
		"form.Value, 'fakeip_range'",
		"form.Flag, 'resolve_for_subnet_rules'",
		"form.Flag, 'nft_counters'",
		"uci.set('neto', 'main', 'fakeip_enabled', '1')",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("advanced.js missing %q:\n%s", want, s)
		}
	}
	for _, forbidden := range []string{
		"form.Flag, 'enabled'",
		"form.Flag, 'fakeip_enabled'",
		"form.Value, 'real_dns_upstream'",
		"form.Value, 'singbox_dns'",
		"form.ListValue, 'dns_upstream_protocol'",
		"form.ListValue, 'real_dns_mode'",
		"form.ListValue, 'real_dns_transport'",
		"form.ListValue, 'routing_mode'",
		"form.ListValue, 'default_outbound'",
	} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("advanced.js should not expose %q:\n%s", forbidden, s)
		}
	}
}

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
		"form.Value, 'dns_listen'",
		"form.ListValue, 'dns_upstream_preset'",
		"form.ListValue, 'dns_upstream_protocol'",
		"form.Value, 'dns_upstream_host'",
		"form.Value, 'dns_upstream_port'",
		"form.Value, 'dns_upstream_tls_name'",
		"form.Value, 'dns_upstream_path'",
		"port = defaultDNSPort(protocol)",
		"uci.set('neto', 'main', 'real_dns_upstream', host + ':' + port)",
		"form.ListValue, 'routing_mode'",
		"form.ListValue, 'default_outbound'",
		"form.ListValue, 'language'",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("general.js missing %q:\n%s", want, s)
		}
	}
	for _, forbidden := range []string{
		"form.Flag, 'enabled'",
		"form.Flag, 'fakeip_enabled'",
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
		"form.Flag, 'filter_aaaa_for_fakeip'",
		"form.DynamicList, 'lan_subnet'",
		"form.DynamicList, 'lan_iface'",
		"form.Value, 'singbox_bin'",
		"form.Value, 'singbox_dns'",
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
		"form.Value, 'dns_listen'",
		"form.Value, 'real_dns_upstream'",
		"form.ListValue, 'dns_upstream_preset'",
		"form.ListValue, 'dns_upstream_protocol'",
		"form.ListValue, 'routing_mode'",
		"form.ListValue, 'default_outbound'",
	} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("advanced.js should not expose %q:\n%s", forbidden, s)
		}
	}
}

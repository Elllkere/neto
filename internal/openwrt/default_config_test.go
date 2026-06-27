package openwrt

import (
	"os"
	"strings"
	"testing"
)

func TestEmbeddedDefaultConfigHasNoSampleClientsOrRules(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/etc/config/neto")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, forbidden := range []string{
		"config client",
		"config rule",
		"gaming_pc",
		"all_proxy",
		"youtube",
		"list lan_subnet '192.168.8.0/24'",
	} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("default config should not include test fixture %q:\n%s", forbidden, s)
		}
	}
	for _, want := range []string{
		"option fakeip_enabled '1'",
		"option dns_upstream_preset 'cloudflare'",
		"option dns_upstream_protocol 'udp'",
		"option dns_upstream_host '1.1.1.1'",
		"option dns_upstream_tls_name 'cloudflare-dns.com'",
		"option dns_upstream_path '/dns-query'",
		"option language 'en'",
		"option language_ru_installed '0'",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("default config missing %q:\n%s", want, s)
		}
	}
}

func TestInstallerDetectsLANSubnetAndConfiguresLanguage(t *testing.T) {
	data, err := os.ReadFile("../../embedded/install.sh")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		"--language en|ru",
		"install Russian LuCI localization",
		"LANGUAGE_CHOICE=\"ru\"",
		"uci set neto.main.language='ru'",
		"uci set neto.main.language_ru_installed='1'",
		"ip -4 route show dev br-lan scope link",
		"ipcalc.sh \"$ipaddr\" \"$netmask\"",
		"ensure_lan_subnet_config",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("installer missing %q:\n%s", want, s)
		}
	}
}

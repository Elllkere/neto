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
		"option real_dns_mode 'direct'",
		"option real_dns_transport 'udp'",
		"option real_dns_server '1.1.1.1:53'",
		"option real_dns_server_name 'cloudflare-dns.com'",
		"option real_dns_path '/dns-query'",
		"option singbox_dns_fakeip '127.0.0.1:15353'",
		"option singbox_dns_real_direct '127.0.0.1:15354'",
		"option singbox_dns_real_proxy '127.0.0.1:15355'",
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
		"network_from_ip_prefix",
		"normalized=\"$(network_from_ip_prefix \"$ipaddr\" \"$prefix\"",
		"ensure_lan_subnet_config",
		"ensure_builtin_providers",
		"https://www.cloudflare.com/ips-v4/",
		"https://core.telegram.org/resources/cidr.txt",
		"provider_url_exists \"$url\"",
		"provider_script_exists \"$script_path\"",
		"ensure_builtin_script_provider \"akamai_ipv4\" \"Akamai IPv4\" \"/usr/share/neto/providers/akamai-ipv4.sh\" \"15\"",
		"ensure_builtin_script_provider \"aws_ipv4\" \"AWS IPv4\" \"/usr/share/neto/providers/aws-ipv4.sh\" \"20\"",
		"uci set \"neto.$section.source=script\"",
		"uci set \"neto.$section.auto_update=0\"",
		"chmod 0755 /usr/share/neto/run-sing-box-log.sh",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("installer missing %q:\n%s", want, s)
		}
	}
}

func TestEmbeddedSingBoxLogWrapperIsInstalledAsset(t *testing.T) {
	path := "../../embedded/files/usr/share/neto/run-sing-box-log.sh"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Fatalf("sing-box log wrapper is not executable: %s", path)
	}
	s := string(data)
	for _, want := range []string{
		"#!/bin/sh",
		"/var/log/neto/sing-box.log",
		"tail -c \"$log_keep_bytes\"",
		"exec \"$bin\" run -c \"$config\" >> \"$log_file\" 2>&1",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("%s missing %q:\n%s", path, want, s)
		}
	}
}

func TestEmbeddedProviderScriptsAreInstalledAssets(t *testing.T) {
	for _, path := range []string{
		"../../embedded/files/usr/share/neto/providers/akamai-ipv4.sh",
		"../../embedded/files/usr/share/neto/providers/aws-ipv4.sh",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0111 == 0 {
			t.Fatalf("provider script is not executable: %s", path)
		}
		s := string(data)
		for _, want := range []string{
			"#!/bin/sh",
			"NETO_PROVIDER_OUTPUT",
			"curl -fsSL",
			"command -v jq",
		} {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q:\n%s", path, want, s)
			}
		}
		if strings.Contains(s, "select(test(") {
			t.Fatalf("%s must not use jq regex functions; OpenWrt jq may be built without ONIGURUMA:\n%s", path, s)
		}
	}
}

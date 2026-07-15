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
		"chmod 0755 /usr/share/neto/run-sing-box-log.sh",
		"chmod 0755 /usr/share/neto/check-version.sh",
		"curl_usable()",
		"wget -O \"$tmp\" \"$url\"",
		"curl -fsSL \"$url\" -o \"$tmp\"",
		"attempts=\"$attempts broken-curl\"",
		"check_runtime_curl",
		"warning: /usr/bin/curl is installed but cannot start",
		"atomic_install()",
		"mv -f \"$tmp\" \"$dest\"",
		"atomic_install \"$WORK_DIR/bin/$arch/netod\" /usr/bin/netod",
		"existing installation detected; preserving config and skipping package installation",
		"verify_installed_version",
		"restart_luci_deferred",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("installer missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "neto.$section.enabled") {
		t.Fatalf("installer must not write provider enabled fields:\n%s", s)
	}
	for _, forbidden := range []string{
		"ensure_builtin_providers",
		"ensure_builtin_provider",
		"ensure_builtin_script_provider",
		"provider_url_exists",
		"provider_script_exists",
		"uci set \"neto.$section=provider\"",
	} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("installer must not auto-create provider sections %q:\n%s", forbidden, s)
		}
	}
}

func TestVersionCheckWrapperCannotTriggerUpgrade(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/usr/share/neto/check-version.sh")
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat("../../embedded/files/usr/share/neto/check-version.sh")
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Fatal("check-version.sh is not executable")
	}
	s := string(data)
	if !strings.Contains(s, "exec /usr/share/neto/upgrade.sh --check") {
		t.Fatalf("version wrapper must force check-only mode:\n%s", s)
	}
}

func TestUpgradeScriptFallsBackAroundBrokenCurl(t *testing.T) {
	data, err := os.ReadFile("../../embedded/upgrade.sh")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		"curl_usable()",
		"--check) MODE=\"check\"",
		"--luci) MODE=\"luci\"",
		"latest_version()",
		"neto-version.txt",
		"RELEASE_API_URL",
		"status=\"available\"",
		"printf 'current=%s\\nlatest=%s\\nstatus=%s\\n'",
		"sh \"$TMP\"",
		"wget -O \"$tmp\" \"$url\"",
		"curl -fsSL \"$url\" -o \"$tmp\"",
		"attempts=\"$attempts broken-curl\"",
		"download \"$INSTALL_URL\" \"$TMP\"",
		"NETO_EXPECT_VERSION=\"$expected\" sh \"$TMP\"",
		"neto upgrade: verified installed version $actual",
		"UPGRADE_LOG=\"${NETO_UPGRADE_LOG:-/tmp/neto/upgrade.log}\"",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("upgrade script missing %q:\n%s", want, s)
		}
	}
}

func TestInstallerRefreshesLuCIAfterUpdate(t *testing.T) {
	data, err := os.ReadFile("../../embedded/install.sh")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		"--no-ui-restart)",
		"clear_luci_cache()",
		"rm -f /tmp/luci-indexcache /var/run/luci-indexcache",
		"rm -rf /tmp/luci-modulecache",
		"/etc/init.d/rpcd restart",
		"/etc/init.d/uhttpd restart",
		"sleep 2",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("installer missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "RESTART_UI=0") {
		t.Fatalf("installer must not skip LuCI restart after replacing views and ACLs:\n%s", s)
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
		"/tmp/neto/sing-box.log",
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
		"../../embedded/files/usr/share/neto/providers/aws-full-ipv4.sh",
		"../../embedded/files/usr/share/neto/providers/aws-full-eu-ipv4.sh",
		"../../embedded/files/usr/share/neto/providers/google-cloud-eu-ipv4.sh",
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

func TestEmbeddedGoogleCloudProviderScriptFiltersEuropeanIPv4Ranges(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/usr/share/neto/providers/google-cloud-eu-ipv4.sh")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		"https://www.gstatic.com/ipranges/cloud.json",
		"startswith(\"europe-\")",
		"/^europe-/",
		".ipv4Prefix // empty",
		"Google Cloud IPv4 ranges for Europe",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("Google Cloud Europe provider missing %q:\n%s", want, s)
		}
	}
}

func TestEmbeddedAWSProviderScriptsAreSplitByService(t *testing.T) {
	cdnData, err := os.ReadFile("../../embedded/files/usr/share/neto/providers/aws-ipv4.sh")
	if err != nil {
		t.Fatal(err)
	}
	fullData, err := os.ReadFile("../../embedded/files/usr/share/neto/providers/aws-full-ipv4.sh")
	if err != nil {
		t.Fatal(err)
	}
	cdn := string(cdnData)
	full := string(fullData)

	for _, want := range []string{"CLOUDFRONT", "S3", "AWS CDN IPv4"} {
		if !strings.Contains(cdn, want) {
			t.Fatalf("AWS CDN provider missing %q:\n%s", want, cdn)
		}
	}
	for _, want := range []string{"AMAZON", "EC2", "GLOBALACCELERATOR", "AWS Full IPv4", "may affect ping to games hosted on Amazon/AWS servers"} {
		if !strings.Contains(full, want) {
			t.Fatalf("AWS Full provider missing %q:\n%s", want, full)
		}
	}
}

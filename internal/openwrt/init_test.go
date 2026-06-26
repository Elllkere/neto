package openwrt

import (
	"os"
	"strings"
	"testing"
)

func TestInitScriptStartsTwoProcdInstances(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/etc/init.d/neto")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if strings.Count(s, "procd_open_instance ") < 2 {
		t.Fatalf("expected at least two procd instances:\n%s", s)
	}
	if !strings.Contains(s, "procd_open_instance netod") {
		t.Fatalf("missing netod procd instance:\n%s", s)
	}
	if !strings.Contains(s, "procd_open_instance sing-box") {
		t.Fatalf("missing sing-box procd instance:\n%s", s)
	}
	if !strings.Contains(s, `"$singbox_bin" check -c /tmp/neto/sing-box.json`) {
		t.Fatalf("missing sing-box check before procd start:\n%s", s)
	}
}

func TestInitScriptManagesDNSMasqUCI(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/etc/init.d/neto")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		"uci -q del_list dhcp.@dnsmasq[0].server=\"$server\"",
		"uci add_list dhcp.@dnsmasq[0].server=\"$server\"",
		"uci set dhcp.@dnsmasq[0].noresolv='1'",
		"uci set dhcp.@dnsmasq[0].addsubnet='32'",
		"uci commit dhcp",
		"DNSMASQ_NORESOLV_STATE",
		"DNSMASQ_ADDSUBNET_STATE",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in init script:\n%s", want, s)
		}
	}
}

package openwrt

import (
	"os"
	"strings"
	"testing"
)

func TestLuCIMenuOrderAndDebugPage(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/usr/share/luci/menu.d/luci-app-neto.json")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		`"admin/services/neto/general"`,
		`"title": "General"`,
		`"order": 10`,
		`"admin/services/neto/rules"`,
		`"title": "Rules"`,
		`"order": 30`,
		`"admin/services/neto/clients"`,
		`"title": "Clients"`,
		`"order": 40`,
		`"admin/services/neto/outbounds"`,
		`"title": "Outbounds"`,
		`"order": 20`,
		`"admin/services/neto/advanced"`,
		`"title": "Advanced"`,
		`"path": "neto/advanced"`,
		`"admin/services/neto/logs"`,
		`"title": "Logs"`,
		`"order": 80`,
		`"path": "neto/logs"`,
		`"admin/services/neto/debug"`,
		`"title": "Debug"`,
		`"order": 90`,
		`"path": "neto/debug"`,
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("menu missing %q:\n%s", want, s)
		}
	}
	for _, forbidden := range []string{
		`"admin/services/neto/overview"`,
		`"title": "Overview"`,
		`"path": "neto/overview"`,
	} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("overview menu entry should be replaced by Debug, found %q:\n%s", forbidden, s)
		}
	}

	if _, err := os.Stat("../../embedded/files/www/luci-static/resources/view/neto/debug.js"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("../../embedded/files/www/luci-static/resources/view/neto/logs.js"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("../../embedded/files/www/luci-static/resources/view/neto/overview.js"); !os.IsNotExist(err) {
		t.Fatalf("overview.js should be removed, stat err=%v", err)
	}
}

func TestLuCIACLAllowsStatusAndVersionCommands(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/usr/share/rpcd/acl.d/luci-app-neto.json")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		`"/bin/pidof": [ "exec" ]`,
		`"/etc/init.d/neto": [ "exec" ]`,
		`"/usr/bin/netod": [ "exec" ]`,
		`"/usr/bin/sing-box": [ "exec" ]`,
		`"/usr/libexec/neto/sing-box": [ "exec" ]`,
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("ACL missing %q:\n%s", want, s)
		}
	}
}

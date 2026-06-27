package openwrt

import (
	"os"
	"strings"
	"testing"
)

func TestLuCIConfigPagesRestartNetoAfterCommit(t *testing.T) {
	for _, path := range []string{
		"../../embedded/files/www/luci-static/resources/view/neto/general.js",
		"../../embedded/files/www/luci-static/resources/view/neto/clients.js",
		"../../embedded/files/www/luci-static/resources/view/neto/rules.js",
		"../../embedded/files/www/luci-static/resources/view/neto/providers.js",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		s := string(data)
		for _, want := range []string{
			"'require fs'",
			"'require uci'",
			"handleSaveApply: function(ev)",
			"return uci.apply();",
			"fs.exec('/etc/init.d/neto', [ 'restart' ])",
			"window.location.reload();",
		} {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q:\n%s", path, want, s)
			}
		}
		for _, forbidden := range []string{
			"on_after_commit",
			"on_before_commit",
		} {
			if strings.Contains(s, forbidden) {
				t.Fatalf("%s must not use unsupported hook %q:\n%s", path, forbidden, s)
			}
		}
	}
}

func TestLuCIACLAllowsNetoRestart(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/usr/share/rpcd/acl.d/luci-app-neto.json")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, `"/etc/init.d/neto": [ "exec" ]`) {
		t.Fatalf("ACL must allow neto restart from LuCI:\n%s", s)
	}
}

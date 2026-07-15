package openwrt

import (
	"os"
	"strings"
	"testing"
)

func TestLuCIConfigPagesRestartNetoAfterCommit(t *testing.T) {
	helperData, err := os.ReadFile("../../embedded/files/www/luci-static/resources/neto/ui.js")
	if err != nil {
		t.Fatal(err)
	}
	helper := string(helperData)
	for _, want := range []string{
		"applyAndRestart: applyAndRestart",
		"showApplyProgress();",
		"ui.showModal(_('Save & Apply')",
		"_('Applying configuration changes…')",
		"_('Configuration changes applied.')",
		"_('Failed to apply configuration changes.')",
		"showApplyError(err);",
		"return uci.apply()",
		"window.setTimeout(resolve, 2500)",
		"fs.exec('/etc/init.d/neto', [ 'restart' ])",
		"return ui.changes.init();",
		"window.location.reload();",
	} {
		if !strings.Contains(helper, want) {
			t.Fatalf("neto/ui.js missing %q:\n%s", want, helper)
		}
	}
	if strings.Contains(helper, "fs.exec('/sbin/uci', [ 'commit', 'neto' ])") {
		t.Fatalf("Save & Apply must not commit outside the LuCI UCI session:\n%s", helper)
	}

	for _, path := range []string{
		"../../embedded/files/www/luci-static/resources/view/neto/general.js",
		"../../embedded/files/www/luci-static/resources/view/neto/advanced.js",
		"../../embedded/files/www/luci-static/resources/view/neto/clients.js",
		"../../embedded/files/www/luci-static/resources/view/neto/outbounds.js",
		"../../embedded/files/www/luci-static/resources/view/neto/rules.js",
		"../../embedded/files/www/luci-static/resources/view/neto/providers.js",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		s := string(data)
		for _, want := range []string{
			"handleSaveApply: function(ev)",
			"return netoUI.applyAndRestart();",
		} {
			if !strings.Contains(s, want) {
				t.Fatalf("%s missing %q:\n%s", path, want, s)
			}
		}
		for _, forbidden := range []string{
			"on_after_commit",
			"on_before_commit",
			"return netoUI.commitAndRestart();",
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
	if strings.Count(s, `"/usr/share/neto/check-version.sh": [ "exec" ]`) != 1 ||
		strings.Count(s, `"/usr/share/neto/upgrade.sh": [ "exec" ]`) != 1 {
		t.Fatalf("ACL must separate read-only version checks from authenticated neto upgrades:\n%s", s)
	}
}

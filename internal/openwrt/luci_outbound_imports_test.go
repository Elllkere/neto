package openwrt

import (
	"os"
	"strings"
	"testing"
)

func TestOutboundsLuCIContainsImportAndSubscriptions(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/outbounds.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		"showImportModal: function()",
		"var importButton, cancelButton;",
		"var status = E('span'",
		"importButton.disabled = true;",
		"cancelButton.disabled = true;",
		"textarea.disabled = true;",
		"importButton.textContent = _('Importing...')",
		"status.style.display = '';",
		"status.style.display = 'none';",
		"handleManualImport: function(value)",
		"fs.write(importPath",
		"fs.exec('/usr/bin/netod', [ 'import-uri', '-file', importPath ])",
		"el.appendChild(E('button'",
		"}, _('Import')))",
		"form.GridSection, 'subscription', _('Subscriptions')",
		"handleSubscriptionUpdate: function(section_id)",
		"return this.handleSave()",
		"fs.exec('/sbin/uci', [ 'commit', 'neto' ])",
		"throw new Error(res.stderr || res.stdout || _('Commit failed'))",
		"fs.exec('/usr/bin/netod', [ 'subscriptions', 'update', section_id ])",
		"form.Value, 'url'",
		"form.Flag, 'auto_update'",
		"form.ListValue, 'update_hour'",
		"for (var hour = 0; hour < 24; hour++)",
		"o.value(String(hour), _('%d:00').format(hour))",
		"o.depends('auto_update', '1')",
		"form.ListValue, 'update_via'",
		"o.value('direct', _('Direct'))",
		"o.value('proxy', _('Proxy'))",
		"form.ListValue, 'update_outbound'",
		"o.depends('update_via', 'proxy')",
		"form.Button, '_update'",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("outbounds.js missing import/subscription UI %q:\n%s", want, s)
		}
	}
	for _, forbidden := range []string{
		"form.GridSection, 'outbound', _('Subscription nodes')",
		"uci.get('neto', section_id, 'subscription') != null",
		"uci.get('neto', section_id, 'subscription') == null",
		"handleSubscriptionUpdate: function(section_id) {\n\t\treturn this.handleSave()\n\t\t\t.then(function() {\n\t\t\t\treturn uci.apply();",
	} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("subscription nodes must remain editable in the regular Outbounds table, found %q:\n%s", forbidden, s)
		}
	}
	for _, want := range []string{
		"vless://",
		"hysteria2://",
		"ss://",
		"trojan://",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("import modal should mention %q:\n%s", want, s)
		}
	}
}

func TestNoSeparateImportsLuCIMenu(t *testing.T) {
	menu, err := os.ReadFile("../../embedded/files/usr/share/luci/menu.d/luci-app-neto.json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(menu), `"admin/services/neto/imports"`) || strings.Contains(string(menu), `"path": "neto/imports"`) {
		t.Fatalf("imports must be integrated into Outbounds, not a separate menu page:\n%s", menu)
	}

	acl, err := os.ReadFile("../../embedded/files/usr/share/rpcd/acl.d/luci-app-neto.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"/tmp/neto-import.txt": [ "read", "write" ]`,
		`"/sbin/uci": [ "exec" ]`,
		`"/usr/bin/netod": [ "exec" ]`,
		`"/etc/init.d/neto": [ "exec" ]`,
	} {
		if !strings.Contains(string(acl), want) {
			t.Fatalf("ACL missing %q:\n%s", want, acl)
		}
	}
}

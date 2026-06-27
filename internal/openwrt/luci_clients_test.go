package openwrt

import (
	"os"
	"strings"
	"testing"
)

func TestClientsLuCIPolicyHelpIsSectionDescription(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/clients.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	help := "Default follows general routing mode. Proxy forces non-reserved traffic through neto. Direct bypasses neto completely."

	if !strings.Contains(s, "form.GridSection, 'client', _('Clients'),") || !strings.Contains(s, "_('"+help+"')") {
		t.Fatalf("clients policy help should be on the Clients section like Rules help:\n%s", s)
	}
	if strings.Contains(s, "form.ListValue, 'policy', _('Policy'),") {
		t.Fatalf("clients policy help must not render inside the table option:\n%s", s)
	}
}

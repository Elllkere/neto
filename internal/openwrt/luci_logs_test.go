package openwrt

import (
	"os"
	"strings"
	"testing"
)

func TestLuCILogsPageUsesNetodLogCommand(t *testing.T) {
	data, err := os.ReadFile("../../embedded/files/www/luci-static/resources/view/neto/logs.js")
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{
		"'require fs'",
		"'require ui'",
		"fs.exec('/usr/bin/netod', args)",
		"netod([ 'logs', 'sing-box' ])",
		"fs.exec('/usr/bin/netod', [ 'logs', 'sing-box', 'clear' ])",
		"handleRefresh: function()",
		"handleClear: function(button)",
		"sing-box Logs",
		"Refresh",
		"Clear",
		"No logs yet",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("logs.js missing %q:\n%s", want, s)
		}
	}
}

package status

import "testing"

func TestLocalRouteStatusMissingTable(t *testing.T) {
	got := localRouteStatusResult("Error: ipv4: FIB table does not exist.\nDump terminated\n", 2, true)
	if got != "missing" {
		t.Fatalf("got %q, want missing", got)
	}
}

func TestLocalRouteStatusPresent(t *testing.T) {
	got := localRouteStatusResult("local default dev lo scope host\n", 0, false)
	if got != "present" {
		t.Fatalf("got %q, want present", got)
	}
}

func TestListenerPresentBusyBoxNetstat(t *testing.T) {
	output := `
Active Internet connections (only servers)
Proto Recv-Q Send-Q Local Address           Foreign Address         State       PID/Program name
tcp        0      0 127.0.0.1:15353         0.0.0.0:*               LISTEN      123/sing-box
udp        0      0 127.0.0.1:5353          0.0.0.0:*                           124/netod
udp        0      0 127.0.0.1:16001         0.0.0.0:*                           123/sing-box
`
	for _, addr := range []string{"127.0.0.1:15353", "127.0.0.1:5353", "127.0.0.1:16001"} {
		if !listenerPresent(output, addr) {
			t.Fatalf("expected %s to be present", addr)
		}
	}
}

func TestListenerPresentSS(t *testing.T) {
	output := `
Netid State  Recv-Q Send-Q Local Address:Port Peer Address:Port
udp   UNCONN 0      0          127.0.0.1:5353      0.0.0.0:*
tcp   LISTEN 0      4096       127.0.0.1:15353     0.0.0.0:*
`
	if !listenerPresent(output, "127.0.0.1:5353") {
		t.Fatal("expected UDP listener")
	}
	if !listenerPresent(output, "127.0.0.1:15353") {
		t.Fatal("expected TCP listener")
	}
	if listenerPresent(output, "127.0.0.1:16001") {
		t.Fatal("unexpected listener")
	}
}

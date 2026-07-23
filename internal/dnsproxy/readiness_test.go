package dnsproxy

import (
	"context"
	"encoding/binary"
	"net"
	"strings"
	"testing"
	"time"
)

func TestWaitReadyCompletesAfterEndToEndDNSResponse(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	go func() {
		buf := make([]byte, 512)
		n, addr, readErr := conn.ReadFrom(buf)
		if readErr != nil {
			return
		}
		response := append([]byte(nil), buf[:n]...)
		response[2] |= 0x80
		response[3] = 0
		_, _ = conn.WriteTo(response, addr)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := WaitReady(ctx, conn.LocalAddr().String()); err != nil {
		t.Fatal(err)
	}
}

func TestWaitReadyReportsTimeout(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := conn.LocalAddr().String()
	_ = conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	err = WaitReady(ctx, address)
	if err == nil || !strings.Contains(err.Error(), "not ready") {
		t.Fatalf("got %v, want readiness timeout", err)
	}
}

func TestReadinessQueryUsesExampleComA(t *testing.T) {
	msg := readinessQuery(0x1234)
	if got := binary.BigEndian.Uint16(msg[0:2]); got != 0x1234 {
		t.Fatalf("query ID=%x", got)
	}
	query, ok := ParseQuery(msg)
	if !ok || query.Name != "example.com" || query.Type != qTypeA {
		t.Fatalf("unexpected readiness query: %+v, parsed=%t", query, ok)
	}
}

package server

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestResolveIPLiteral(t *testing.T) {
	got, err := resolveIP("127.0.0.1")
	if err != nil {
		t.Fatalf("resolveIP() error = %v", err)
	}
	if got != "127.0.0.1" {
		t.Fatalf("resolveIP() = %q, want 127.0.0.1", got)
	}
}

func TestTCPPingLocalListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	accepted := make(chan struct{})
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			_ = conn.Close()
		}
		close(accepted)
	}()

	latency, err := tcpPing(listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("tcpPing() error = %v", err)
	}
	if latency < 0 {
		t.Fatalf("tcpPing() latency = %d, want >= 0", latency)
	}

	select {
	case <-accepted:
	case <-time.After(time.Second):
		t.Fatal("local TCP listener did not accept the probe")
	}
}

func TestHTTPPingLocalServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	latency, err := httpPing(server.URL, time.Second)
	if err != nil {
		t.Fatalf("httpPing() error = %v", err)
	}
	if latency < 0 {
		t.Fatalf("httpPing() latency = %d, want >= 0", latency)
	}
}

func TestHTTPPingRejectsErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "failure", http.StatusInternalServerError)
	}))
	defer server.Close()

	if _, err := httpPing(server.URL, time.Second); err == nil || !strings.Contains(err.Error(), "http status not ok") {
		t.Fatalf("httpPing() error = %v, want HTTP status error", err)
	}
}

func TestICMPPingIntegration(t *testing.T) {
	if os.Getenv("KOMARIX_RUN_PRIVILEGED_NETWORK_TESTS") != "1" {
		t.Skip("set KOMARIX_RUN_PRIVILEGED_NETWORK_TESTS=1 to run raw-socket ICMP integration test")
	}

	latency, err := icmpPing("127.0.0.1", time.Second)
	if err != nil {
		t.Fatalf("icmpPing() error = %v", err)
	}
	if latency < 0 {
		t.Fatalf("icmpPing() latency = %d, want >= 0", latency)
	}
}

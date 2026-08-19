package config

import "testing"

func TestParseUsesDefaultAddress(t *testing.T) {
	cfg, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}

	if cfg.Addr != ":6379" {
		t.Fatalf("expected default address :6379, got %q", cfg.Addr)
	}
}

func TestParseAcceptsAddress(t *testing.T) {
	cfg, err := Parse([]string{"-addr", "127.0.0.1:6380"})
	if err != nil {
		t.Fatalf("Parse returned an error: %v", err)
	}

	if cfg.Addr != "127.0.0.1:6380" {
		t.Fatalf("expected custom address, got %q", cfg.Addr)
	}
}

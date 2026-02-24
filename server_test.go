package main

import (
	"testing"
)

func TestNewServeCmd(t *testing.T) {
	cmd := newServeCmd()
	if cmd == nil {
		t.Fatal("expected command, got nil")
	}
	if cmd.Use != "serve" {
		t.Errorf("expected Use 'serve', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description")
	}
	if cmd.Run == nil {
		t.Error("expected Run function")
	}
}

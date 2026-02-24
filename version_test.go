package main

import (
	"testing"
)

func TestNewVersionCmd(t *testing.T) {
	cmd := newVersionCmd()
	if cmd == nil {
		t.Fatal("expected command, got nil")
	}
	if cmd.Use != "version" {
		t.Errorf("expected Use 'version', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description")
	}
	if cmd.Run == nil {
		t.Error("expected Run function")
	}
}

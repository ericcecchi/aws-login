package main

import (
	"testing"

	"golang.org/x/term"
)

func TestChooseInteractiveEmpty(t *testing.T) {
	_, err := chooseInteractive([]string{}, func(s string) string { return s })
	if err == nil {
		t.Fatalf("expected error for empty items")
	}
}

func TestChooseInteractiveRequiresTTY(t *testing.T) {
	if term.IsTerminal(0) {
		t.Skip("stdin is a terminal; cannot assert non-tty behavior")
	}
	_, err := chooseInteractive([]string{"a"}, func(s string) string { return s })
	if err == nil {
		t.Fatalf("expected tty error")
	}
}

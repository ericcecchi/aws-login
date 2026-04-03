package awslogin

import (
	"testing"

	"golang.org/x/term"
)

func TestChooseInteractiveEmpty(t *testing.T) {
	_, err := chooseInteractive([]string{}, "Test", func(s string) string { return s })
	if err == nil {
		t.Fatalf("expected error for empty items")
	}
}

func TestChooseInteractiveRequiresTTY(t *testing.T) {
	if term.IsTerminal(0) {
		t.Skip("stdin is a terminal; cannot assert non-tty behavior")
	}
	_, err := chooseInteractive([]string{"a"}, "Test", func(s string) string { return s })
	if err == nil {
		t.Fatalf("expected tty error")
	}
}

func TestPickerWindowSize(t *testing.T) {
	tests := []struct {
		total int
		want  int
	}{
		{0, 0},
		{1, 1},
		{4, 4},
		{5, 5},
		{8, 8},
		{12, 12},
		{13, 12},
		{100, 12},
	}
	for _, tt := range tests {
		got := pickerWindowSize(tt.total)
		if got != tt.want {
			t.Errorf("pickerWindowSize(%d) = %d, want %d", tt.total, got, tt.want)
		}
	}
}

func TestFilterItem(t *testing.T) {
	items := []string{
		"prod-account (123456789012)",
		"dev-account (987654321098)",
		"staging-account (555555555555)",
	}

	tests := []struct {
		input string
		index int
		want  bool
		desc  string
	}{
		// Empty input matches everything — ensures all items show on initial render.
		{"", 0, true, "empty matches first"},
		{"", 1, true, "empty matches second"},
		{"", 2, true, "empty matches third"},
		// Whitespace-only input also matches everything.
		{"   ", 0, true, "whitespace matches first"},
		// Substring match (case-insensitive).
		{"prod", 0, true, "prod matches prod-account"},
		{"prod", 1, false, "prod does not match dev-account"},
		{"PROD", 0, true, "PROD case-insensitive match"},
		{"account", 0, true, "account matches all"},
		{"account", 2, true, "account matches staging too"},
		// Account ID fragment.
		{"1234", 0, true, "partial account ID match"},
		{"1234", 1, false, "partial account ID no match"},
		// No match.
		{"xyzzy", 0, false, "no match"},
		// Out-of-bounds index returns false, not panic.
		{"prod", -1, false, "negative index safe"},
		{"prod", 99, false, "out-of-bounds index safe"},
	}

	for _, tt := range tests {
		got := filterItem(items, tt.input, tt.index)
		if got != tt.want {
			t.Errorf("filterItem(%q, %d) [%s] = %v, want %v", tt.input, tt.index, tt.desc, got, tt.want)
		}
	}
}


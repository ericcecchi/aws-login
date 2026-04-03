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

package awslogin

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"golang.org/x/term"
)

// filterItem reports whether the display label at the given index matches the
// search input.  An empty input matches every item so the initial list is fully
// visible.  Exported as a named function so it can be unit-tested independently
// of the terminal UI.
func filterItem(displayItems []string, input string, index int) bool {
	needle := strings.TrimSpace(strings.ToLower(input))
	if needle == "" {
		return true
	}
	if index < 0 || index >= len(displayItems) {
		return false
	}
	return strings.Contains(strings.ToLower(displayItems[index]), needle)
}

func chooseInteractive[T any](items []T, title string, label func(T) string) (T, error) {
	var zero T
	if len(items) == 0 {
		return zero, fmt.Errorf("no options available")
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return zero, fmt.Errorf("interactive selection requires a TTY; provide --account and --role or use --non-interactive")
	}
	if title == "" {
		title = "Select"
	}

	displayItems := make([]string, 0, len(items))
	for _, item := range items {
		displayItems = append(displayItems, label(item))
	}

	searcher := func(input string, index int) bool {
		return filterItem(displayItems, input, index)
	}

	// The shell integration wrapper captures aws-login's stdout with $() so
	// that it can eval the exported AWS_PROFILE.  Writing the interactive UI
	// to stdout would make it invisible to the user.  Stderr is never captured
	// by the wrapper and is always connected to the user's terminal, so we
	// direct all picker output there.
	selector := promptui.Select{
		Label:            fmt.Sprintf("%s", title),
		Items:            displayItems,
		Size:             pickerWindowSize(len(displayItems)),
		Searcher:         searcher,
		StartInSearchMode: true,
		Stdout:           os.Stderr,
		Templates: &promptui.SelectTemplates{
			Label:    `{{ "▶" | cyan }} {{ . | bold }}`,
			Active:   `  {{ "›" | cyan }} {{ . | cyan }}`,
			Inactive: `    {{ . }}`,
			Selected: `  {{ "✓" | green }} {{ . | green }}`,
		},
	}

	idx, _, err := selector.Run()
	if err != nil {
		if errors.Is(err, promptui.ErrInterrupt) || errors.Is(err, promptui.ErrEOF) {
			return zero, fmt.Errorf("selection cancelled")
		}
		return zero, fmt.Errorf("selection failed: %w", err)
	}
	return items[idx], nil
}

func pickerWindowSize(total int) int {
	const minSize = 5
	const maxSize = 12
	if total < minSize {
		return total
	}
	if total > maxSize {
		return maxSize
	}
	return total
}


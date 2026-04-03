package awslogin

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"golang.org/x/term"
)

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
		needle := strings.TrimSpace(strings.ToLower(input))
		if needle == "" {
			return true
		}
		return strings.Contains(strings.ToLower(displayItems[index]), needle)
	}

	selector := promptui.Select{
		Label:    fmt.Sprintf("%s  (type to filter, enter to choose)", title),
		Items:    displayItems,
		Size:     pickerWindowSize(len(displayItems)),
		Searcher: searcher,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   `▸ {{ . | cyan }}`,
			Inactive: `  {{ . }}`,
			Selected: `✓ {{ . | green }}`,
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

package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/ktr0731/go-fuzzyfinder"
	"golang.org/x/term"
)

func chooseInteractive[T any](items []T, label func(T) string) (T, error) {
	var zero T
	if len(items) == 0 {
		return zero, fmt.Errorf("no options available")
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return zero, fmt.Errorf("interactive selection requires a TTY; provide --account and --role or use --non-interactive")
	}

	idx, err := fuzzyfinder.Find(
		items,
		func(i int) string { return label(items[i]) },
		fuzzyfinder.WithPromptString("Select> "),
	)
	if err != nil {
		if errors.Is(err, fuzzyfinder.ErrAbort) {
			return zero, fmt.Errorf("selection cancelled")
		}
		return zero, fmt.Errorf("selection failed")
	}
	return items[idx], nil
}

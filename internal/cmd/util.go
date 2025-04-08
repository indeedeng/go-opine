package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

// executeNoArgs checks that no positional arguments were passed, and
// if so runs impl().
//
// If arguments were provided a usage error will be written and
// subcommands.ExitUsageError will be returned.
//
// If impl() returns an error it is written to f.Output() and
// subcommands.ExitFailure is returned. Otherwise subcommands.ExitSuccess
// is returned.
func executeNoArgs(f *flag.FlagSet, impl func() error) subcommands.ExitStatus {
	if !ensureNoArgs(f) {
		return subcommands.ExitUsageError
	}
	if err := impl(); err != nil {
		_, _ = fmt.Fprintln(f.Output(), err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

// ensureNoArgs checks that no positional arguments were provided.
func ensureNoArgs(f *flag.FlagSet) bool {
	if f.NArg() == 0 {
		return true
	}

	_, _ = fmt.Fprintln(
		f.Output(),
		fmt.Errorf("unexpected positional argument(s): %q", f.Args()))
	f.Usage()
	return false
}

// closedTempFile creates a temp file, closes it, and returns the file path.
func closedTempFile(dir, pattern string) (string, error) {
	tmpCov, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	if err := tmpCov.Close(); err != nil {
		return "", err
	}
	return tmpCov.Name(), nil
}

package cmd

import (
	"errors"
	"strings"
)

var (
	// errCoverageCheckFailed is returned by the "test" subcommand when the
	// minimum coverage requirements are not met.
	errCoverageCheckFailed = errors.New("coverage check failed")

	// errNoTests is returned by the "test" subcommand when there are no tests.
	errNoTests = errors.New("no tests")
)

func CombineErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	var sb strings.Builder
	sb.WriteString("multiple errors occurred:\n")
	for _, err := range errs {
		sb.WriteString("  * " + err.Error() + "\n")
	}
	return errors.New(sb.String())
}

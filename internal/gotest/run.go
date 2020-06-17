package gotest

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
)

// Option can be passed to Run to change how it behaves (e.g. test
// with -race, or write verbose output somewhere).
type Option func(o *options) error

type options struct {
	race         bool
	coverprofile string
	coverpkg     string
	covermode    string
	p            int
	accepters    []resultAccepter
}

// Race runs tests with -race.
func Race() Option {
	return func(o *options) error {
		o.race = true
		return nil
	}
}

// CoverProfile runs tests with -coverprofile=<path>.
func CoverProfile(path string) Option {
	return func(o *options) error {
		o.coverprofile = path
		return nil
	}
}

// CoverPkg runs tests with -coverpkg=<patterns>.
func CoverPkg(patterns string) Option {
	return func(o *options) error {
		o.coverpkg = patterns
		return nil
	}
}

// CoverMode runs tests with -covermode=<mode>.
func CoverMode(mode string) Option {
	return func(o *options) error {
		o.covermode = mode
		return nil
	}
}

// P runs tests with -p=<p>. This controls the number of test binaries
// that can be run in parallel.
//
// See the -p option of "go help build" for more information.
func P(p int) Option {
	return func(o *options) error {
		if p <= 0 {
			return fmt.Errorf("gotest: invalid option -p: %q", p)
		}
		o.p = p
		return nil
	}
}

// QuietOutput writes output similar to "go test" (without "-v")
// to the provided writer.
func QuietOutput(to io.Writer) Option {
	return func(o *options) error {
		o.accepters = append(o.accepters, &quietOutput{to: to})
		return nil
	}
}

// QuietOutput writes output similar to "go test -v" to the
// provided writer.
func VerboseOutput(to io.Writer) Option {
	return func(o *options) error {
		o.accepters = append(o.accepters, &verboseOutput{to: to})
		return nil
	}
}

// Run runs go test.
func Run(opts ...Option) error {
	var o options
	for _, opt := range opts {
		if err := opt(&o); err != nil {
			return err
		}
	}

	// Now build the command to run with the realized options
	// struct.
	args := []string{"test", "-v", "-json"}
	if o.race {
		args = append(args, "-race")
	}
	if o.coverprofile != "" {
		args = append(args, "-coverprofile="+o.coverprofile)
	}
	if o.coverpkg != "" {
		args = append(args, "-coverpkg="+o.coverpkg)
	}
	if o.covermode != "" {
		args = append(args, "-covermode="+o.covermode)
	}
	if o.p != 0 {
		args = append(args, "-p="+strconv.Itoa(o.p))
	}
	args = append(args, "./...")
	cmd := exec.Command("go", args...)
	cmd.Stderr = os.Stderr

	cmdStdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := parseGoTestJSONOutput(cmdStdout, newMultiResultAccepter(o.accepters...)); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("go test failed: %w", err)
	}

	return nil
}

func parseGoTestJSONOutput(r io.Reader, to resultAccepter) error {
	grouper := newResultPackageGrouper(to)
	aggregator := newResultAggregator(newRemoveCoverageOutput(newWorkaroundGoIssue35180ResultAccepter(grouper)))
	parser := newEventStreamParser(aggregator)
	if err := parser.Parse(r); err != nil {
		return err
	}
	if err := aggregator.CheckAllEventsConsumed(); err != nil {
		return err
	}
	if err := grouper.CheckAllResultsConsumed(); err != nil {
		return err
	}
	return nil
}

package cmd

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"

	"github.com/google/subcommands"

	"oss.indeed.com/go/go-opine/internal/coverage"
	"oss.indeed.com/go/go-opine/internal/gotest"
	"oss.indeed.com/go/go-opine/internal/junit"
)

const (
	defaultMinCoverage = 50.0
)

// hasATestRegexp will match any "go test" output that has at least one
// test. This is done by checking for any line that starts with "=== RUN".
var hasATestRegexp = regexp.MustCompile(`(?m)^=== RUN\b`)

// TestCmd returns a subcommand that tests a go project.
func TestCmd() subcommands.Command {
	return &testCmd{
		out:           os.Stdout,
		minCovPercent: defaultMinCoverage,
	}
}

type testCmd struct {
	out io.Writer

	junit         string
	xmlcov        string
	coverprofile  string
	norace        bool
	minCovPercent float64
}

func (*testCmd) Name() string {
	return "test"
}

func (*testCmd) Synopsis() string {
	return "run Go tests in an opinionated way"
}

func (*testCmd) Usage() string {
	return `test [-min-coverage <percent>] [-junit <path>] [-xmlcov <path>] [-coverprofile <path>]:
  Run Go tests in an opinionated way.
`
}

func (t *testCmd) SetFlags(f *flag.FlagSet) {
	f.Float64Var(&t.minCovPercent, "min-coverage", defaultMinCoverage, "minimum code test coverage to enforce")
	f.StringVar(&t.junit, "junit", "", "write JUnit XML test results")
	f.StringVar(&t.xmlcov, "xmlcov", "", "write Cobertura XML coverage")
	f.StringVar(&t.coverprofile, "coverprofile", "", "write Go coverprofile coverage")
	f.BoolVar(&t.norace, "norace", false, "compile tests with race detector disabled")
}

//revive:disable:unused-parameter
func (t *testCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	return executeNoArgs(f, t.impl)
}

func (t *testCmd) impl() error {
	covPath, err := closedTempFile("", "go-opine-coverprofile.")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for coverprofile output: %w", err)
	}

	var errs []error
	var testOutBuf bytes.Buffer
	options := []gotest.Option{
		gotest.Race(),
		gotest.CoverProfile(covPath),
		gotest.CoverPkg("./..."),
		gotest.CoverMode("atomic"),
		gotest.P(runtime.GOMAXPROCS(0)),
		gotest.QuietOutput(t.out),
		gotest.VerboseOutput(&testOutBuf),
	}
	if !t.norace {
		options = append(options, gotest.Race())
	}

	testErr := gotest.Run(options...)
	if testErr != nil {
		errs = append(errs, fmt.Errorf("unit tests failed: %w", testErr))
	}

	testOut := testOutBuf.String()
	if !hasATestRegexp.MatchString(testOut) {
		errs = append(errs, errNoTests)
		return CombineErrors(errs)
	}

	if t.junit != "" {
		if junitErr := junit.Write(testOut, t.junit); junitErr != nil {
			errs = append(errs, fmt.Errorf("failed to write JUnit XML: %w", junitErr))
		}
	}

	if cov, covLoadErr := coverage.Load(covPath); covLoadErr == nil {
		if t.xmlcov != "" {
			if xmlCovErr := cov.XML(t.xmlcov); xmlCovErr != nil {
				errs = append(errs, fmt.Errorf("failed to write XML coverage: %w", xmlCovErr))
			}
		}
		if t.coverprofile != "" {
			if covProfileErr := cov.CoverProfile(t.coverprofile); covProfileErr != nil {
				errs = append(errs, fmt.Errorf("failed to write coverprofile coverage: %w", covProfileErr))
			}
		}

		covRatio := cov.Ratio()
		if covRatio < t.minCovPercent/100 {
			_, _ = fmt.Printf(
				"Insufficient test coverage (%.1f%% < %.1f%%).\nSet the -min-coverage flag to configure coverage requirements.\n",
				covRatio*100,
				t.minCovPercent,
			)
			errs = append(errs, errCoverageCheckFailed)
		} else {
			_, _ = fmt.Printf(
				"Test coverage sufficient (%.1f%% >= %.1f%%)\n",
				covRatio*100,
				t.minCovPercent,
			)
		}
	} else {
		errs = append(errs, fmt.Errorf("failed to load coverage: %w", covLoadErr))
	}

	return CombineErrors(errs)
}

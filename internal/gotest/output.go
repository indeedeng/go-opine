package gotest

import (
	"io"
	"regexp"
)

var (
	removeCoverageOutputRegexp = regexp.MustCompile(`(?m)(?:^|\n|\t)coverage:[^\t\n]*`)
	removePassOutputRegexp     = regexp.MustCompile(`(?m)(?:\nPASS$|^PASS\n)`)
)

const testFailure = "fail"

// removeCoverageOutput is a resultAccepter that removes coverage-related
// output from results before forwarding to the next result accepter.
type removeCoverageOutput struct {
	next resultAccepter
}

var _ resultAccepter = (*removeCoverageOutput)(nil)

func newRemoveCoverageOutput(next resultAccepter) *removeCoverageOutput {
	return &removeCoverageOutput{next: next}
}

func (r *removeCoverageOutput) Accept(res result) error {
	res.Output = removeCoverageOutputRegexp.ReplaceAllString(res.Output, "")
	return r.next.Accept(res)
}

// verboseOutput is a resultAccepter that writes "go test -v"-like
// output to an io.Writer.
type verboseOutput struct {
	to io.Writer
}

var _ resultAccepter = (*verboseOutput)(nil)

func newVerboseOutput(to io.Writer) *verboseOutput {
	return &verboseOutput{to: to}
}

func (v *verboseOutput) Accept(res result) error {
	_, err := v.to.Write([]byte(res.Output))
	return err
}

// quietOutput is a resultAccepter that writes "go test"-like (no "-v")
// output to an io.Writer.
type quietOutput struct {
	to io.Writer
}

var _ resultAccepter = (*quietOutput)(nil)

func newQuietOutput(to io.Writer) *quietOutput {
	return &quietOutput{to: to}
}

func (q quietOutput) Accept(res result) error {
	// Print output from failed tests.
	if res.Key.Test != "" && res.Outcome == testFailure {
		_, err := q.to.Write([]byte(res.Output))
		return err
	}
	// Print output from build output
	if res.Key.ImportPath != "" {
		_, err := q.to.Write([]byte(res.Output))
		return err
	}

	// Remove "PASS" lines from the non-test (i.e. package) output. There
	// is already a line starting with "?", "ok", or "fail" that indicates the
	// package result.
	if res.Key.Test == "" {
		_, err := q.to.Write([]byte(removePassOutputRegexp.ReplaceAllString(res.Output, "")))
		return err
	}

	return nil
}

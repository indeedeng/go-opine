package gotest

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var workaroundGoIssue35180Regexp = regexp.MustCompile(`\n((?:exit status \d+\n)?FAIL\s+(\S+)\s+(\S+)(?:\s+coverage:[^\n]+)?\n)$`)

const (
	workaroundGoIssue35180RegexpOutput  = 1
	workaroundGoIssue35180RegexpPackage = 2
	workaroundGoIssue35180RegexpElapsed = 3
)

// workaroundGoIssue35180ResultAccepter detects when a failed
// test result ends with output normally associated with a package
// result and injects the package result. This is to work around Go
// issue https://github.com/golang/go/issues/35180.
type workaroundGoIssue35180ResultAccepter struct {
	next resultAccepter
}

var _ resultAccepter = (*workaroundGoIssue35180ResultAccepter)(nil)

func newWorkaroundGoIssue35180ResultAccepter(next resultAccepter) resultAccepter {
	return &workaroundGoIssue35180ResultAccepter{next: next}
}

func (w *workaroundGoIssue35180ResultAccepter) Accept(res result) error {
	if res.Outcome != "fail" || res.Key.Test == "" {
		return w.next.Accept(res)
	}

	match := workaroundGoIssue35180Regexp.FindStringSubmatch(res.Output)
	if match == nil || match[workaroundGoIssue35180RegexpPackage] != res.Key.Package {
		return w.next.Accept(res)
	}
	matchOutput := match[workaroundGoIssue35180RegexpOutput]
	matchElapsed := match[workaroundGoIssue35180RegexpElapsed]

	res.Output = res.Output[:len(res.Output)-len(matchOutput)]
	if err := w.next.Accept(res); err != nil {
		return err
	}

	pkgRes := result{
		Key:     resultKey{Package: res.Key.Package},
		Outcome: "fail",
		Output:  matchOutput,
	}
	if strings.HasSuffix(matchElapsed, "s") {
		if elapsedSecs, err := strconv.ParseFloat(matchElapsed[:len(matchElapsed)-1], 64); err == nil {
			pkgRes.Elapsed = time.Duration(elapsedSecs * float64(time.Second))
		}
	}
	return w.next.Accept(pkgRes)
}

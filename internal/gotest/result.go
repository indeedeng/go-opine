package gotest

import (
	"fmt"
	"strings"
	"time"
)

// resultKey identifies a result.
type resultKey struct {
	Package string
	Test    string
	// as of go 1.24 build output is categorized by ImportPath json events
	ImportPath string
}

// result is a test result. The result is for either a single test or for
// a package, in which case Key.Test is empty.
type result struct {
	Key     resultKey
	Outcome string
	Output  string
	Elapsed time.Duration
}

// resultAccepter accepts results.
type resultAccepter interface {
	Accept(res result) error
}

// multiResultAccepter accepts results and forwards them on to zero or
// more downstream result accepters.
type multiResultAccepter struct {
	accepters []resultAccepter
}

var _ resultAccepter = (*multiResultAccepter)(nil)

func newMultiResultAccepter(accepter ...resultAccepter) *multiResultAccepter {
	return &multiResultAccepter{accepters: accepter}
}

// Accept forwards the result to the downstream resultAccepters. If any
// resultAccepter returns an error processing stops immediately and that
// error is returned to the caller.
func (m multiResultAccepter) Accept(res result) error {
	for _, accepter := range m.accepters {
		if err := accepter.Accept(res); err != nil {
			return err
		}
	}
	return nil
}

// resultAggregator is an eventAccepter that aggregates events for the same
// test or package into results. Completed results are passed to the
// resultAccepter.
type resultAggregator struct {
	to     resultAccepter
	events map[resultKey][]event
	err    error
}

func newResultAggregator(to resultAccepter) *resultAggregator {
	return &resultAggregator{
		to:     to,
		events: make(map[resultKey][]event),
	}
}

// Accept adds an event to the internal state and provides any result
// completed by the event to the resultAccepter.
//
// If the resultAccepter returns an error the resultAggregator will enter
// an error state causing the current accept and all subsequent accepts to
// fail. This error will also be returned by Close.
func (a *resultAggregator) Accept(e event) error {
	if a.err != nil {
		return fmt.Errorf("permanent error state: %w", a.err)
	}

	rk := resultKey{
		Package:    e.Package,
		Test:       e.Test,
		ImportPath: e.ImportPath,
	}

	if !isTestOrPackageComplete(e.Action) {
		a.events[rk] = append(a.events[rk], e)
		return nil
	}

	var output strings.Builder
	for _, prevEvent := range a.events[rk] {
		output.WriteString(prevEvent.Output)
	}
	delete(a.events, rk)
	output.WriteString(e.Output)

	res := result{
		Key:     rk,
		Outcome: e.Action,
		Output:  output.String(),
		Elapsed: time.Duration(e.Elapsed * float64(time.Second)),
	}
	if err := a.to.Accept(res); err != nil {
		a.setErr(err)
		return a.err
	}

	return nil
}

// filterBuildWarnings returns a copy of the supplied events map after removing
// any resultKeys that only contain "build-output" actions. These actions
// without a cooresponding build-fail event are just build warnings and do not
// change the output of the tests.
func filterBuildWarnings(events map[resultKey][]event) map[resultKey][]event {
	filtered := make(map[resultKey][]event, len(events))
	for rk, eventsByRK := range events {
		foundEvent := false
		for _, event := range eventsByRK {
			if event.Action != "build-output" {
				foundEvent = true
				break
			}
		}
		if foundEvent {
			filtered[rk] = events[rk]
		}
	}
	return filtered
}

// CheckAllEventsConsumed checks that all events are consumed and that
// no error occurred in any Accept.
func (a *resultAggregator) CheckAllEventsConsumed() error {
	filteredEvents := filterBuildWarnings(a.events)
	if a.err == nil && len(filteredEvents) > 0 {
		a.setErr(fmt.Errorf("not all events were consumed: %#v", filteredEvents))
	}
	return a.err
}

// setErr puts the resultAggregator into a permanent error state.
func (a *resultAggregator) setErr(err error) {
	a.err = err
	a.events = nil
}

// resultPackageGrouper accepts results, groups them by package, and
// forwards all results for a package when it completes.
//
// This is necessary because by default Go will run tests from different
// packages at the same time. If the output of each result is printed
// immediately it will cause confusion regarding which package each test
// is in. For example take the following output:
//
//	=== RUN   Test_Cmd_optionLog
//	--- PASS: Test_Cmd_optionLog (0.01s)
//	PASS
//	ok  	oss.indeed.com/go/go-opine/internal/run	(cached)
//
// The only way you can tell the Test_Cmd_optionLog package is
// oss.indeed.com/go/go-opine/internal/run is by the fact that the package
// output is printed immediately after the test output.
//
// !!WARNING!! This struct relies on the the final result of a package
// being the "package result" (i.e. the result that has only a package
// and no test). If you filter results before providing them to a
// resultPackageGrouper make sure you do not filter out the package result
// for any test result you previously provided. Otherwise Close will return
// an error about results remaining.
type resultPackageGrouper struct {
	to         resultAccepter
	pkgResults map[string][]result
	err        error
}

var _ resultAccepter = (*resultPackageGrouper)(nil)

func newResultPackageGrouper(to resultAccepter) *resultPackageGrouper {
	return &resultPackageGrouper{
		to:         to,
		pkgResults: make(map[string][]result),
	}
}

// Accept adds the result to the resultPackageGrouper internal state and,
// if the result is a "package result", forwards all buffered test results
// and the package result onward.
//
// If the resultAccepter returns an error the resultPackageGrouper will enter
// an error state causing the current accept and all subsequent accepts to
// fail. This error will also be returned by Close.
func (r *resultPackageGrouper) Accept(res result) error {
	if r.err != nil {
		return fmt.Errorf("permanent error state: %w", r.err)
	}

	r.pkgResults[res.Key.Package] = append(r.pkgResults[res.Key.Package], res)

	if !isPackageComplete(res) {
		return nil
	}

	if err := r.forward(r.pkgResults[res.Key.Package]...); err != nil {
		return err
	}
	delete(r.pkgResults, res.Key.Package)

	return nil
}

// CheckAllEventsConsumed checks that all results are consumed and that
// no error occurred in any Accept.
func (r *resultPackageGrouper) CheckAllResultsConsumed() error {
	if r.err == nil && len(r.pkgResults) > 0 {
		r.setErr(fmt.Errorf("not all results were consumed: %#v", r.pkgResults))
	}
	return r.err
}

// forward passes zero or more results on to the resultAccepter. If the
// resultAccepter returns an error for any result processing stops, setErr
// is called to put the resultPackageGrouper in a permanent error state, and
// the error is returned.
func (r *resultPackageGrouper) forward(results ...result) error {
	for _, res := range results {
		if err := r.to.Accept(res); err != nil {
			r.setErr(err)
			return r.err
		}
	}
	return nil
}

// setErr puts the resultPackageGrouper into a permanent error state.
func (r *resultPackageGrouper) setErr(err error) {
	r.err = err
	r.pkgResults = nil
}

// isTestOrPackageComplete returns true iff the provided event.Action
// represents the completion of test or package.
func isTestOrPackageComplete(action string) bool {
	return action == "pass" || action == "fail" || action == "skip" || action == "build-fail"
}

// isPackageComplete returns true iff the provided result represents
// the completion of a package.
func isPackageComplete(res result) bool {
	return res.Key.Test == ""
}

package gotest

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// resultKey identifies a result.
type resultKey struct {
	Package string
	Test    string
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
	to              resultAccepter
	eventsByPackage map[string]*packageEvents
	err             error
}

type packageEvents struct {
	eventsByTest map[string][]event
	events       []event
	latestEvent  time.Time
}

func newResultAggregator(to resultAccepter) *resultAggregator {
	return &resultAggregator{
		to:              to,
		eventsByPackage: make(map[string]*packageEvents),
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

	if _, packageKnown := a.eventsByPackage[e.Package]; !packageKnown {
		a.eventsByPackage[e.Package] = &packageEvents{
			eventsByTest: make(map[string][]event),
		}
	}
	packageEvents := a.eventsByPackage[e.Package]

	if packageEvents.latestEvent.IsZero() || e.Time.After(packageEvents.latestEvent) {
		packageEvents.latestEvent = e.Time
	}

	if e.Test == "" {
		packageEvents.events = append(packageEvents.events, e)
		if actionComplete(e.Action) {
			return a.permanentError(a.packageComplete(e.Package))
		}
		return nil
	}

	packageEvents.eventsByTest[e.Test] = append(packageEvents.eventsByTest[e.Test], e)
	if actionComplete(e.Action) {
		return a.permanentError(a.testComplete(e.Package, e.Test))
	}

	return nil
}

// NoMoreEvents notifies the aggregator no more events will be
// submitted via Accept. Generates synthetic results for tests/packages
// it did not see a final event for.
func (a *resultAggregator) NoMoreEvents() error {
	if a.err != nil {
		return a.err
	}

	for p := range a.eventsByPackage {
		err := a.packageComplete(p)
		if err != nil {
			return a.permanentError(err)
		}
	}

	return nil
}

func (a *resultAggregator) packageComplete(packageName string) error {
	packageEvents := a.eventsByPackage[packageName]
	for t := range packageEvents.eventsByTest {
		err := a.testComplete(packageName, t)
		if err != nil {
			return err
		}
	}

	delete(a.eventsByPackage, packageName)

	return a.to.Accept(renderResultImpl(packageName, "", packageEvents.events, packageEvents.latestEvent))
}

func (a *resultAggregator) testComplete(packageName, testName string) error {
	packageEvents := a.eventsByPackage[packageName]
	testEvents := packageEvents.eventsByTest[testName]

	delete(packageEvents.eventsByTest, testName)

	return a.to.Accept(renderResultImpl(packageName, testName, testEvents, time.Time{}))
}

// permanentError puts the resultAggregator into a permanent error state.
func (a *resultAggregator) permanentError(err error) error {
	if err != nil {
		a.err = err
		a.eventsByPackage = nil
	}
	return err
}

func renderResultImpl(
	packageName string,
	testName string,
	events []event,
	latestPackageEvent time.Time,
) result {
	var firstEventTime time.Time
	var lastEventTime time.Time
	var output strings.Builder
	for _, e := range events {
		if firstEventTime.IsZero() || e.Time.Before(firstEventTime) {
			firstEventTime = e.Time
		}
		if lastEventTime.IsZero() || e.Time.After(lastEventTime) {
			lastEventTime = e.Time
		}
		_, _ = output.WriteString(e.Output)
	}

	synthetic := true
	outcome := testFailure
	elapsed := lastEventTime.Sub(firstEventTime)
	if len(events) != 0 {
		lastEvent := events[len(events)-1]
		if actionComplete(lastEvent.Action) {
			synthetic = false
			outcome = lastEvent.Action
			if lastEvent.Elapsed > 0 {
				elapsed = time.Duration(lastEvent.Elapsed * float64(time.Second))
			}
		}
	}

	if synthetic {
		// mimicking go test's output
		if testName == "" {
			elapsed = latestPackageEvent.Sub(firstEventTime)
			_, _ = output.WriteString("FAIL\n")
			_, _ = output.WriteString("FAIL\t")
			_, _ = output.WriteString(packageName)
			_, _ = output.WriteString("\t")
			_, _ = output.WriteString(fmt.Sprintf("%.02f", elapsed.Seconds()))
			_, _ = output.WriteString("s\n")
		} else {
			_, _ = output.WriteString("--- FAIL: ")
			_, _ = output.WriteString(testName)
			_, _ = output.WriteString(" (")
			_, _ = output.WriteString(fmt.Sprintf("%.02f", elapsed.Seconds()))
			_, _ = output.WriteString("s)\n")
		}
	}

	return result{
		Key:     resultKey{Package: packageName, Test: testName},
		Outcome: outcome,
		Output:  output.String(),
		Elapsed: elapsed,
	}
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

	if res.Key.Test != "" {
		return nil
	}

	if err := r.forward(r.pkgResults[res.Key.Package]...); err != nil {
		return err
	}
	delete(r.pkgResults, res.Key.Package)

	return nil
}

// CheckAllResultsConsumed checks that all results are consumed and that
// no error occurred in any Accept.
func (r *resultPackageGrouper) CheckAllResultsConsumed() error {
	if r.err == nil && len(r.pkgResults) > 0 {
		r.setErr(errors.New("not all results were consumed"))
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

// actionComplete returns true iff the provided event.Action
// represents the completion of test or package.
func actionComplete(action string) bool {
	return action == "pass" || action == "fail" || action == "skip"
}

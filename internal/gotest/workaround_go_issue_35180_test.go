package gotest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_workaroundGoIssue35180ResultAccepter_Accept(t *testing.T) {
	issueRes := result{
		Key:     resultKey{Package: "example.com", Test: "Test_Panic"},
		Outcome: "fail",
		Output: `panic: runtime error: OMG SO BROKEN
... stack
    trace ...
FAIL	example.com	3.452s
`,
	}
	var results []result
	tested := newWorkaroundGoIssue35180ResultAccepter(
		resultAccepterFunc(func(res result) error { results = append(results, res); return nil }),
	)
	require.NoError(t, tested.Accept(issueRes))
	require.Equal(
		t,
		[]result{
			{
				Key:     issueRes.Key,
				Outcome: issueRes.Outcome,
				Output: `panic: runtime error: OMG SO BROKEN
... stack
    trace ...
`,
			},
			{
				Key:     resultKey{Package: issueRes.Key.Package},
				Outcome: issueRes.Outcome,
				Output: "FAIL	example.com	3.452s\n",
				Elapsed: 3452 * time.Millisecond,
			},
		},
		results,
	)
}

func Test_workaroundGoIssue35180ResultAccepter_Accept_withExitStatusAndCoverage(t *testing.T) {
	issueRes := result{
		Key:     resultKey{Package: "example.com", Test: "Test_Panic"},
		Outcome: "fail",
		Output: `panic: runtime error: OMG SO BROKEN
... stack
    trace ...
exit status 2
FAIL	example.com	3.452s	coverage: proly lots
`,
	}
	var results []result
	tested := newWorkaroundGoIssue35180ResultAccepter(
		resultAccepterFunc(func(res result) error { results = append(results, res); return nil }),
	)
	require.NoError(t, tested.Accept(issueRes))
	require.Equal(
		t,
		[]result{
			{
				Key:     issueRes.Key,
				Outcome: issueRes.Outcome,
				Output: `panic: runtime error: OMG SO BROKEN
... stack
    trace ...
`,
			},
			{
				Key:     resultKey{Package: issueRes.Key.Package},
				Outcome: issueRes.Outcome,
				Output: `exit status 2
FAIL	example.com	3.452s	coverage: proly lots
`,
				Elapsed: 3452 * time.Millisecond,
			},
		},
		results,
	)
}

func Test_workaroundGoIssue35180ResultAccepter_Accept_passthroughNotFail(t *testing.T) {
	res := result{
		Key:     resultKey{Package: "example.com", Test: "Test_Something"},
		Outcome: "pass",
		Output:  "some output",
		Elapsed: 42 * time.Second,
	}
	var results []result
	tested := newWorkaroundGoIssue35180ResultAccepter(
		resultAccepterFunc(func(res result) error { results = append(results, res); return nil }),
	)
	require.NoError(t, tested.Accept(res))
	require.Equal(t, []result{res}, results)
}

func Test_workaroundGoIssue35180ResultAccepter_Accept_passthroughNormalFailure(t *testing.T) {
	res := result{
		Key:     resultKey{Package: "example.com", Test: "Test_Something"},
		Outcome: "fail",
		Output: `
=== RUN   Test_Something
--- FAIL: Test_Something (0.00s)
`,
		Elapsed: 42 * time.Second,
	}
	var results []result
	tested := newWorkaroundGoIssue35180ResultAccepter(
		resultAccepterFunc(func(res result) error { results = append(results, res); return nil }),
	)
	require.NoError(t, tested.Accept(res))
	require.Equal(t, []result{res}, results)
}

func Test_workaroundGoIssue35180ResultAccepter_Accept_passthroughPackageMismatch(t *testing.T) {
	res := result{
		Key:     resultKey{Package: "example.com/some/pkg", Test: "Test_Weird_Panic"},
		Outcome: "fail",
		Output: `panic: runtime error: OMG SO BROKEN
... stack
    trace ...
FAIL	example.com/other/pkg	3.452s
`,
		Elapsed: 42 * time.Second,
	}
	var results []result
	tested := newWorkaroundGoIssue35180ResultAccepter(
		resultAccepterFunc(func(res result) error { results = append(results, res); return nil }),
	)
	require.NoError(t, tested.Accept(res))
	require.Equal(t, []result{res}, results)
}

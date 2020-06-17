package gotest

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_resultAccepter_Accept(t *testing.T) {
	var (
		firstCalled  bool
		secondCalled bool
	)
	expectedResult := result{Outcome: "BLAH"}
	tested := newMultiResultAccepter(
		resultAccepterFunc(func(res result) error { require.Equal(t, expectedResult, res); firstCalled = true; return nil }),
		resultAccepterFunc(func(res result) error { require.Equal(t, expectedResult, res); secondCalled = true; return nil }),
	)
	require.NoError(t, tested.Accept(expectedResult))
	require.True(t, firstCalled)
	require.True(t, secondCalled)
}

func Test_resultAccepter_Accept_error(t *testing.T) {
	expectedErr := errors.New("fail boat")
	tested := newMultiResultAccepter(
		resultAccepterFunc(func(res result) error { return expectedErr }),
		resultAccepterFunc(func(res result) error { require.Fail(t, "should not be called"); return nil }),
	)
	require.Equal(t, expectedErr, tested.Accept(result{}))
}

func Test_resultAccepter_Accept_nil(t *testing.T) {
	tested := newMultiResultAccepter()
	require.NoError(t, tested.Accept(result{}))
}

func Test_resultAggregator_Accept(t *testing.T) {
	var results []result
	tested := newResultAggregator(
		resultAccepterFunc(func(res result) error { results = append(results, res); return nil }),
	)

	// First test starts running.
	require.NoError(
		t,
		tested.Accept(
			event{
				Action:  "run",
				Package: "indeed.com/some/pkg",
				Test:    "Test_Some",
				Output:  "=== RUN   Test_Some\n",
			},
		),
	)
	require.Empty(t, results)

	// NOTE This test is running in parallel. The previous Test_Some is
	//      still running.
	require.NoError(
		t,
		tested.Accept(
			event{
				Action:  "run",
				Package: "indeed.com/some/other/pkg",
				Test:    "Test_Other",
				Output:  "=== RUN   Test_Other\n",
			},
		),
	)
	require.Empty(t, results)

	// Some output from the first test.
	require.NoError(
		t,
		tested.Accept(
			event{
				Action:  "output",
				Package: "indeed.com/some/pkg",
				Test:    "Test_Some",
				Output:  "Some output\nMultiple lines\nNo trailing newline",
			},
		),
	)
	require.Empty(t, results)

	// More output from the first test.
	require.NoError(
		t,
		tested.Accept(
			event{
				Action:  "run",
				Package: "indeed.com/some/pkg",
				Test:    "Test_Some",
				Output:  "\nThe End\n",
			},
		),
	)
	require.Empty(t, results)

	// Second test completes. The result is passed on.
	require.NoError(
		t,
		tested.Accept(
			event{
				Action:  "pass",
				Package: "indeed.com/some/other/pkg",
				Test:    "Test_Other",
				Output:  "--- PASS: Test_Other (0.42s)\n",
				Elapsed: 0.42,
			},
		),
	)
	require.Equal(
		t,
		[]result{
			{
				Key: resultKey{
					Package: "indeed.com/some/other/pkg",
					Test:    "Test_Other",
				},
				Outcome: "pass",
				Output:  "=== RUN   Test_Other\n--- PASS: Test_Other (0.42s)\n",
				Elapsed: 420 * time.Millisecond,
			},
		},
		results,
	)
	results = nil

	// Second package completes. The result is passed on.
	require.NoError(
		t,
		tested.Accept(
			event{
				Action:  "pass",
				Package: "indeed.com/some/other/pkg",
				Output: "ok  	indeed.com/some/other/pkg	0.091s\n",
				Elapsed: 0.43,
			},
		),
	)
	require.Equal(
		t,
		[]result{
			{
				Key: resultKey{
					Package: "indeed.com/some/other/pkg",
				},
				Outcome: "pass",
				Output: "ok  	indeed.com/some/other/pkg	0.091s\n",
				Elapsed: 430 * time.Millisecond,
			},
		},
		results,
	)
	results = nil

	// Some other package is skipped. The result is passed on.
	require.NoError(
		t,
		tested.Accept(
			event{
				Action:  "skip",
				Package: "indeed.com/some/skipped/pkg",
				Output: "?   	oss.indeed.com/go/go-opine	[no test files]\n",
				Elapsed: 0,
			},
		),
	)
	require.Equal(
		t,
		[]result{
			{
				Key: resultKey{
					Package: "indeed.com/some/skipped/pkg",
				},
				Outcome: "skip",
				Output: "?   	oss.indeed.com/go/go-opine	[no test files]\n",
			},
		},
		results,
	)
	results = nil

	// The first test fails. The result is passed on.
	require.NoError(
		t,
		tested.Accept(
			event{
				Action:  "fail",
				Package: "indeed.com/some/pkg",
				Test:    "Test_Some",
				Output:  "--- PASS: Test_Some (0.02s)\n",
				Elapsed: 0.02,
			},
		),
	)
	require.Equal(
		t,
		[]result{
			{
				Key: resultKey{
					Package: "indeed.com/some/pkg",
					Test:    "Test_Some",
				},
				Outcome: "fail",
				Output:  "=== RUN   Test_Some\nSome output\nMultiple lines\nNo trailing newline\nThe End\n--- PASS: Test_Some (0.02s)\n",
				Elapsed: 20 * time.Millisecond,
			},
		},
		results,
	)
	results = nil

	// The first package fails. The result is passed on.
	require.NoError(
		t,
		tested.Accept(
			event{
				Action:  "fail",
				Package: "indeed.com/some/pkg",
				Output: "FAIL	indeed.com/some/pkg	4.321s\n",
				Elapsed: 4.321,
			},
		),
	)
	require.Equal(
		t,
		[]result{
			{
				Key: resultKey{
					Package: "indeed.com/some/pkg",
				},
				Outcome: "fail",
				Output: "FAIL	indeed.com/some/pkg	4.321s\n",
				Elapsed: 4321 * time.Millisecond,
			},
		},
		results,
	)
	results = nil

	require.NoError(t, tested.CheckAllEventsConsumed())
}

func Test_resultAggregator_Accept_error(t *testing.T) {
	expectedErr := errors.New("fail boat")
	called := false
	tested := newResultAggregator(
		resultAccepterFunc(func(res result) error { require.False(t, called); called = true; return expectedErr }),
	)
	require.Equal(t, expectedErr, tested.Accept(event{Action: "pass"}))
	require.Equal(t, expectedErr, errors.Unwrap(tested.Accept(event{Action: "pass"})))
	require.Equal(t, expectedErr, tested.CheckAllEventsConsumed())
}

func Test_resultAggregator_CheckAllEventsConsumed_unconsumedEvents(t *testing.T) {
	tested := newResultAggregator(resultAccepterFunc(func(res result) error { return nil }))
	require.NoError(t, tested.Accept(event{Package: "BLAH"}))
	require.Error(t, tested.CheckAllEventsConsumed())
}

func Test_resultPackageGrouper_Accept_onePackageNoTests(t *testing.T) {
	var results []result
	tested := newResultPackageGrouper(
		resultAccepterFunc(func(res result) error { results = append(results, res); return nil }),
	)

	packageRes := result{
		Key: resultKey{
			Package: "MyPackage",
		},
	}
	require.NoError(t, tested.Accept(packageRes))
	require.Equal(t, []result{packageRes}, results)
	results = nil

	require.NoError(t, tested.CheckAllResultsConsumed())
}

func Test_resultPackageGrouper_Accept_OnePackageOneTest(t *testing.T) {
	var results []result
	tested := newResultPackageGrouper(
		resultAccepterFunc(func(res result) error { results = append(results, res); return nil }),
	)

	testRes := result{
		Key: resultKey{
			Package: "MyPackage",
			Test:    "MyTest",
		},
	}
	require.NoError(t, tested.Accept(testRes))
	require.Empty(t, results)

	packageRes := result{
		Key: resultKey{
			Package: "MyPackage",
		},
	}
	require.NoError(t, tested.Accept(packageRes))
	require.Equal(t, []result{testRes, packageRes}, results)
	results = nil

	require.NoError(t, tested.CheckAllResultsConsumed())
}

func Test_resultPackageGrouper_Accept_overlappingPackages(t *testing.T) {
	var results []result
	tested := newResultPackageGrouper(
		resultAccepterFunc(func(res result) error { results = append(results, res); return nil }),
	)

	pkg1Test1Res := result{
		Key: resultKey{
			Package: "Package1",
			Test:    "Test1",
		},
	}
	require.NoError(t, tested.Accept(pkg1Test1Res))
	require.Empty(t, results)

	pkg2Test1Res := result{
		Key: resultKey{
			Package: "Package2",
			Test:    "Test1",
		},
	}
	require.NoError(t, tested.Accept(pkg2Test1Res))
	require.Empty(t, results)

	pkg3Test1Res := result{
		Key: resultKey{
			Package: "Package3",
			Test:    "Test1",
		},
	}
	require.NoError(t, tested.Accept(pkg3Test1Res))
	require.Empty(t, results)

	pkg1Res := result{
		Key: resultKey{
			Package: "Package1",
		},
	}
	require.NoError(t, tested.Accept(pkg1Res))
	require.Equal(t, []result{pkg1Test1Res, pkg1Res}, results)
	results = nil

	pkg2Res := result{
		Key: resultKey{
			Package: "Package2",
		},
	}
	require.NoError(t, tested.Accept(pkg2Res))
	require.Equal(t, []result{pkg2Test1Res, pkg2Res}, results)
	results = nil

	pkg3Res := result{
		Key: resultKey{
			Package: "Package3",
		},
	}
	require.NoError(t, tested.Accept(pkg3Res))
	require.Equal(t, []result{pkg3Test1Res, pkg3Res}, results)
	results = nil

	require.NoError(t, tested.CheckAllResultsConsumed())
}

func Test_resultPackageGrouper_Accept_firstPackageCompletesLast(t *testing.T) {
	var results []result
	tested := newResultPackageGrouper(
		resultAccepterFunc(func(res result) error { results = append(results, res); return nil }),
	)

	pkg1Test1Res := result{
		Key: resultKey{
			Package: "Package1",
			Test:    "Test1",
		},
	}
	require.NoError(t, tested.Accept(pkg1Test1Res))
	require.Empty(t, results)

	pkg2Test1Res := result{
		Key: resultKey{
			Package: "Package2",
			Test:    "Test1",
		},
	}
	require.NoError(t, tested.Accept(pkg2Test1Res))
	require.Empty(t, results)

	pkg3Res := result{
		Key: resultKey{
			Package: "Package3",
		},
	}
	require.NoError(t, tested.Accept(pkg3Res))
	require.Equal(t, []result{pkg3Res}, results)
	results = nil

	pkg2Res := result{
		Key: resultKey{
			Package: "Package2",
		},
	}
	require.NoError(t, tested.Accept(pkg2Res))
	require.Equal(t, []result{pkg2Test1Res, pkg2Res}, results)
	results = nil

	pkg1Res := result{
		Key: resultKey{
			Package: "Package1",
		},
	}
	require.NoError(t, tested.Accept(pkg1Res))
	require.Equal(t, []result{pkg1Test1Res, pkg1Res}, results)
	results = nil

	require.NoError(t, tested.CheckAllResultsConsumed())
}

func Test_resultPackageGrouper_Accept_error(t *testing.T) {
	expectedErr := errors.New("fail boat")
	called := false
	tested := newResultPackageGrouper(
		resultAccepterFunc(func(res result) error { require.False(t, called); called = true; return expectedErr }),
	)
	require.Equal(t, expectedErr, tested.Accept(result{}))
	require.Equal(t, expectedErr, errors.Unwrap(tested.Accept(result{})))
	require.Equal(t, expectedErr, tested.CheckAllResultsConsumed())
}

func Test_resultPackageGrouper_CheckAllResultsConsumed_incompletePackages(t *testing.T) {
	tested := newResultPackageGrouper(resultAccepterFunc(func(res result) error { return nil }))
	require.NoError(t, tested.Accept(result{Key: resultKey{Package: "MyPackage", Test: "MyTest"}}))
	require.Error(t, tested.CheckAllResultsConsumed())
}

type resultAccepterFunc func(res result) error

func (f resultAccepterFunc) Accept(res result) error {
	return f(res)
}

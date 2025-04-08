package gotest

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_removeCoverageOutput_Accept(t *testing.T) {
	const output = `PASS
coverage: 67.8% of statements
ok  	oss.indeed.com/go/go-opine/internal/cmd	11.527s	coverage: 67.8% of statements
`
	const expectedOutputWithoutCoverage = `PASS
ok  	oss.indeed.com/go/go-opine/internal/cmd	11.527s
`
	called := false
	tested := newRemoveCoverageOutput(resultAccepterFunc(func(res result) error {
		require.False(t, called)
		require.Equal(t, expectedOutputWithoutCoverage, res.Output)
		called = true
		return nil
	}))
	err := tested.Accept(result{Output: output})
	require.True(t, called)
	require.NoError(t, err)
}

func Test_removeCoverageOutput_Accept_error(t *testing.T) {
	expectedErr := errors.New("failed to parse")
	tested := newRemoveCoverageOutput(resultAccepterFunc(func(result) error { return expectedErr }))
	err := tested.Accept(result{})
	require.Equal(t, expectedErr, err)
}

func Test_verboseOutput_Accept(t *testing.T) {
	const expectedOutput = "Tama-tan no tamashii tamatama Tamachi"
	var output bytes.Buffer
	tested := newVerboseOutput(&output)
	err := tested.Accept(result{Output: expectedOutput})
	require.NoError(t, err)
	require.Equal(t, expectedOutput, output.String())
}

func Test_verboseOutput_Accept_error(t *testing.T) {
	expectedErr := errors.New("failed to parse")
	tested := newVerboseOutput(&errorWriter{err: expectedErr})
	err := tested.Accept(result{Output: "success"})
	require.Equal(t, expectedErr, err)
}

func Test_quietOutput_Accept_ignoresPassingTests(t *testing.T) {
	var output bytes.Buffer
	tested := newQuietOutput(&output)
	err := tested.Accept(
		result{
			Key:     resultKey{Package: "indeed.com/some/package", Test: "Test_Some_test"},
			Outcome: "pass",
			Output:  "You will not see this.",
		},
	)
	require.NoError(t, err)
	require.Empty(t, output.String())
}

func Test_quietOutput_Accept_printsFailingTests(t *testing.T) {
	const expectedOutput = "You will see this."
	var output bytes.Buffer
	tested := newQuietOutput(&output)
	err := tested.Accept(
		result{
			Key:     resultKey{Package: "indeed.com/some/package", Test: "Test_Some_test"},
			Outcome: "fail",
			Output:  expectedOutput,
		},
	)
	require.NoError(t, err)
	require.Equal(t, expectedOutput, output.String())
}

func Test_quietOutput_Accept_printsPassingPackagesWithPASSRemoved(t *testing.T) {
	const outputBeforeRemovingPASS = `PASS
ok  	oss.indeed.com/go/go-opine/internal/cmd	11.527s
`
	const expectedOutputWithPASSRemoved = "ok  	oss.indeed.com/go/go-opine/internal/cmd	11.527s\n"
	var output bytes.Buffer
	tested := newQuietOutput(&output)
	err := tested.Accept(
		result{
			Key:     resultKey{Package: "indeed.com/some/package"},
			Outcome: "pass",
			Output:  outputBeforeRemovingPASS,
		},
	)
	require.NoError(t, err)
	require.Equal(t, expectedOutputWithPASSRemoved, output.String())
}

func Test_quietOutput_Accept_printsFailingPackages(t *testing.T) {
	const expectedOutput = `FAIL
FAIL	oss.indeed.com/go/go-opine/internal/cmd	11.527s
`
	var output bytes.Buffer
	tested := newQuietOutput(&output)
	err := tested.Accept(
		result{
			Key:    resultKey{Package: "indeed.com/some/package"},
			Output: expectedOutput,
		},
	)
	require.NoError(t, err)
	require.Equal(t, expectedOutput, output.String())
}

func Test_quietOutput_Accept_error(t *testing.T) {
	expectedErr := errors.New("failed")
	tested := newQuietOutput(&errorWriter{err: expectedErr})
	err := tested.Accept(result{Output: "success"})
	require.Equal(t, expectedErr, err)
}

type errorWriter struct {
	err error
}

func (e *errorWriter) Write([]byte) (int, error) {
	return 0, e.err
}

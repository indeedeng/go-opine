package gotest

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_P_errorLessThanOne(t *testing.T) {
	tested := P(0)
	require.Error(t, tested(&options{}))
}

func Test_Run_noOptions(t *testing.T) {
	popd := pushd(t, "testdata")
	defer popd()
	require.NoError(t, Run())
}

func Test_Run_allOptions(t *testing.T) {
	const (
		expectedTestOutput    = "Test_Some_test"
		expectedPackage       = "oss.indeed.com/go/go-opine/internal/gotest/testdata"
		expectedPackageOutput = "ok  	" + expectedPackage
	)

	popd := pushd(t, "testdata")
	defer popd()

	f, err := ioutil.TempFile("", "go-opine-gotest.")
	require.NoError(t, err)
	covPath := f.Name()
	require.NoError(t, f.Close())

	var (
		quietOutputBuf   bytes.Buffer
		verboseOutputBuf bytes.Buffer
	)
	err = Run(
		Race(),
		CoverProfile(covPath),
		CoverPkg("./..."),
		CoverMode("atomic"),
		P(1),
		QuietOutput(&quietOutputBuf),
		VerboseOutput(&verboseOutputBuf),
	)
	require.NoError(t, err)
	var (
		quietOutput   = quietOutputBuf.String()
		verboseOutput = verboseOutputBuf.String()
	)
	require.NotContains(t, quietOutput, expectedTestOutput)
	require.Contains(t, verboseOutput, expectedTestOutput)
	require.Contains(t, quietOutput, expectedPackageOutput)
	require.Contains(t, verboseOutput, expectedPackageOutput)

	cov, err := os.ReadFile(covPath)
	require.NoError(t, err)
	require.Contains(t, string(cov), expectedPackage)
}

func Test_Run_fail(t *testing.T) {
	const failTestEnv = "GOTEST_FAIL"
	require.NoError(t, os.Setenv(failTestEnv, "1"))
	defer func() {
		require.NoError(t, os.Unsetenv(failTestEnv))
	}()
	popd := pushd(t, "testdata")
	defer popd()
	err := Run()
	require.Error(t, err)
}

func Test_parseGoTestJSONOutput_parseError(t *testing.T) {
	err := parseGoTestJSONOutput(strings.NewReader("NOT JSON!"), resultAccepterFunc(func(res result) error { return nil }))
	require.Error(t, err)
}

func Test_parseGoTestJSONOutput_unconsumedEvents(t *testing.T) {
	const eventJSON = `{"Time":"2019-09-26T13:27:17.563229183Z","Action":"output","Package":"oss.indeed.com/go/go-opine/internal/cmd","Test":"Test_testCmd_impl","Output":"--- PASS: Test_testCmd_impl (1.93s)\n"}`
	err := parseGoTestJSONOutput(strings.NewReader(eventJSON), resultAccepterFunc(func(res result) error { return nil }))
	require.Error(t, err)
}

func Test_parseGoTestJSONOutput_unconsumedResults(t *testing.T) {
	const eventJSON = `{"Time":"2019-09-26T13:27:17.56324465Z","Action":"pass","Package":"oss.indeed.com/go/go-opine/internal/cmd","Test":"Test_testCmd_impl","Elapsed":1.93}`
	err := parseGoTestJSONOutput(strings.NewReader(eventJSON), resultAccepterFunc(func(res result) error { return nil }))
	require.Error(t, err)
}

func pushd(t *testing.T, path string) func() {
	prev, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(path))
	return func() {
		require.NoError(t, os.Chdir(prev))
	}
}

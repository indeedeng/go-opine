package cmd

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_TestCmd_impl(t *testing.T) {
	popd := pushd(t, "testdata", "go-library")
	defer popd()

	outDir, err := os.MkdirTemp("", "go-opine-cmd-test.")
	require.NoError(t, err)
	defer os.RemoveAll(outDir)

	junitPath := filepath.Join(outDir, "junit.xml")
	xmlcovPath := filepath.Join(outDir, "lang-go-cobertura.xml")
	coverprofilePath := filepath.Join(outDir, "cover.out")
	tested := testCmd{
		out:          io.Discard,
		junit:        junitPath,
		xmlcov:       xmlcovPath,
		coverprofile: coverprofilePath,
	}
	err = tested.impl()
	require.NoError(t, err)

	// Check that the files were generated.
	junitBytes, err := os.ReadFile(junitPath)
	require.NoError(t, err)
	require.Contains(t, string(junitBytes), "\"Test_Library\"")
	xmlcovBytes, err := os.ReadFile(xmlcovPath)
	require.NoError(t, err)
	require.Contains(t, string(xmlcovBytes), "\"library/library.go\"")
	coverProfileBytes, err := os.ReadFile(coverprofilePath)
	require.NoError(t, err)
	require.Contains(t, string(coverProfileBytes), "mode:")
}

func Test_TestCmd_impl_xmlcovWithoutCoverprofile(t *testing.T) {
	popd := pushd(t, "testdata", "go-library")
	defer popd()

	outDir, err := os.MkdirTemp("", "go-opine-cmd-test.")
	require.NoError(t, err)
	defer os.RemoveAll(outDir)

	junitPath := filepath.Join(outDir, "junit.xml")
	xmlcovPath := filepath.Join(outDir, "lang-go-cobertura.xml")
	tested := testCmd{
		out:    io.Discard,
		junit:  junitPath,
		xmlcov: xmlcovPath,
	}
	err = tested.impl()
	require.NoError(t, err)

	// Check that the files were generated.
	junitBytes, err := os.ReadFile(junitPath)
	require.NoError(t, err)
	require.Contains(t, string(junitBytes), "\"Test_Library\"")
	xmlcovBytes, err := os.ReadFile(xmlcovPath)
	require.NoError(t, err)
	require.Contains(t, string(xmlcovBytes), "\"library/library.go\"")
}

func Test_TestCmd_impl_noArgs(t *testing.T) {
	popd := pushd(t, "testdata", "go-library")
	defer popd()

	tested := testCmd{out: io.Discard}
	err := tested.impl()
	require.NoError(t, err)
}

func Test_TestCmd_impl_noTests(t *testing.T) {
	popd := pushd(t, "testdata", "go-kitchen-sink")
	defer popd()

	tested := testCmd{
		out:           io.Discard,
		minCovPercent: 51,
	}
	err := tested.impl()
	require.Error(t, err)
	require.Equal(t, errNoTests, err)
}

func Test_TestCmd_impl_sufficientCoverage(t *testing.T) {
	popd := pushd(t, "testdata", "go-library")
	defer popd()

	tested := testCmd{
		out:           io.Discard,
		minCovPercent: 5, // 50% is sufficient because generated.go is excluded
	}
	err := tested.impl()
	require.NoError(t, err)
}

func Test_TestCmd_impl_insufficientCoverage(t *testing.T) {
	popd := pushd(t, "testdata", "go-library")
	defer popd()

	tested := testCmd{
		out:           io.Discard,
		minCovPercent: 51,
	}
	err := tested.impl()
	require.Error(t, err)
	require.Equal(t, errCoverageCheckFailed, err)
}

func Test_TestCmd_impl_outputsStillWrittenWhenTestsFail(t *testing.T) {
	popd := pushd(t, "testdata", "go-library")
	defer popd()

	outDir, err := os.MkdirTemp("", "go-opine-cmd-test.")
	require.NoError(t, err)
	defer os.RemoveAll(outDir)

	const failTestEnv = "LIBRARY_FAIL_UNIT_TESTS"
	err = os.Setenv(failTestEnv, "1")
	require.NoError(t, err)
	defer os.Unsetenv(failTestEnv)

	junitPath := filepath.Join(outDir, "junit.xml")
	xmlcovPath := filepath.Join(outDir, "lang-go-cobertura.xml")
	coverprofilePath := filepath.Join(outDir, "cover.out")
	tested := testCmd{
		out:          io.Discard,
		junit:        junitPath,
		xmlcov:       xmlcovPath,
		coverprofile: coverprofilePath,
	}
	err = tested.impl()
	require.Error(t, err)

	// Check that the files were generated (even when the tests failed).
	junitBytes, err := os.ReadFile(junitPath)
	require.NoError(t, err)
	require.Contains(t, string(junitBytes), "\"Test_Library\"")
	xmlcovBytes, err := os.ReadFile(xmlcovPath)
	require.NoError(t, err)
	require.Contains(t, string(xmlcovBytes), "\"library/library.go\"")
	coverProfileBytes, err := os.ReadFile(coverprofilePath)
	require.NoError(t, err)
	require.Contains(t, string(coverProfileBytes), "mode:")
}

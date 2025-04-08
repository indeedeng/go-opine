package junit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"oss.indeed.com/go/go-opine/internal/run"
)

func Test_Write(t *testing.T) {
	outDir, err := os.MkdirTemp("", "go-opine-junit-test.")
	require.NoError(t, err)
	defer os.RemoveAll(outDir)
	goTestOutput, _, err := run.Cmd("go", run.Args("test", "-v", "./testdata"))
	require.NoError(t, err)
	outPath := filepath.Join(outDir, "junit.xml")
	err = Write(goTestOutput, outPath)
	require.NoError(t, err)
	outBytes, err := os.ReadFile(outPath)
	require.NoError(t, err)
	out := string(outBytes)
	require.Contains(t, out, "\"Test_Data\"")
}

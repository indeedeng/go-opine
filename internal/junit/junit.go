// Package junit is for writing JUnit XML reports.
package junit

import (
	"io"
	"os"

	"oss.indeed.com/go/go-opine/internal/run"
)

// Write a JUnit XML file from the provided Go test output.
func Write(goTestOutput, outPath string) error {
	junitOut, _, err := run.Cmd(
		"go",
		run.Args("run", "github.com/jstemmer/go-junit-report@v0.9.1"),
		run.Stdin(goTestOutput),
		run.Log(io.Discard),
	)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, []byte(junitOut), 0666); err != nil { //nolint:gosec
		return err
	}
	return nil
}

// Package junit is for writing JUnit XML reports.
package junit

import (
	"io/ioutil"

	"oss.indeed.com/go/go-opine/internal/run"
)

// Write a JUnit XML file from the provided Go test output.
func Write(goTestOutput string, outPath string) error {
	junitOut, _, err := run.Cmd(
		"go",
		run.Args("run", "github.com/jstemmer/go-junit-report"),
		run.Stdin(goTestOutput),
		run.Log(ioutil.Discard),
	)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(outPath, []byte(junitOut), 0666); err != nil {
		return err
	}
	return nil
}

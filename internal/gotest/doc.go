// Package gotest runs "go test -v -json" and interprets the result.
//
// This package contains a LOT of code to
//
//   1. reassemble test events into test results (each test is reported as
//      a bunch of "output" actions followed by a "pass", "fail", or "skip"
//      action), and
//   2. order the results so that when printed each test can be easily
//      associated with the package it is in.
//
// However, all of the above complexity is hidden behind a relatively
// simple API. Here is an example:
//
//     var verboseOutput bytes.Buffer
//     err := gotest.Run(
//         gotest.QuietOutput(os.Stdout),
//         gotest.VerboseOutput(&verboseOutput),
//     )
//
// The above example also shows why we use "go test -v -json" at all: we
// want to be able to construct both the verbose output (for generating
// the JUnit XML) and the quiet output (for printing to the console as the
// tests run).
//
// Note that output is held in memory for buffered results. Since "go test"
// also buffers output this is not likely to be an issue, but if it is we may
// want to consider compressing the output.
package gotest

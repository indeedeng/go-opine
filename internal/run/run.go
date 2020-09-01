// Package run provides utilities for executing external programs.
package run

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"

	"oss.indeed.com/go/go-opine/internal/printing"
)

const (
	outPrefix = "  > "
	errPrefix = "  ! "
)

type cmdinfo struct {
	cmd       *exec.Cmd
	log       *printing.LogWriter
	logStdout bool
	logStderr bool
}

// Cmd runs the provided command (with the provided args) and returns the
// stdout and stderr. A non-nil error will be returned when the command
// exit code is non-zero.
//
// By default Cmd will write the following to os.Stdout:
//   * A message before running the command that indicates the command that
//     will be run, including the args.
//   * The stdout of the command. Each line will be prefixed with "  > ". Note
//     that the returned stdout will not have this prefix.
//   * The stderr of the command. Each line will be prefixed with "  ! ". Note
//     that the returned stderr will not have this prefix.
//   * A message when the command completes indicating whether it completed
//     successfully or not, and what exit code it returned in the case of a
//     failure.
//
// The above output can be configured as follows:
//   * Use Log to change the output destination (from os.Stdout) to the
//     provided io.Writer. Use ioutil.Discard if you do not want anything
//     printed.
//   * Use SuppressStdout to prevent the stdout from being written. It will
//     still be returned.
//   * Use SuppressStderr to prevent the stderr from being written. It will
//     still be returned.
func Cmd(command string, args []string, opts ...Option) (string, string, error) {
	sp := cmdinfo{
		cmd:       exec.Command(command, args...),
		log:       printing.NewLogWriter(os.Stdout),
		logStdout: true,
		logStderr: true,
	}
	for _, opt := range opts {
		opt(&sp)
	}

	var stdout, stderr bytes.Buffer

	stdouts := append(make([]io.Writer, 0, 3), &stdout)
	if sp.cmd.Stdout != nil {
		stdouts = append(stdouts, sp.cmd.Stdout)
	}
	if sp.logStdout {
		stdouts = append(stdouts, printing.NewLinePrefixWriter(sp.log, outPrefix))
	}
	sp.cmd.Stdout = io.MultiWriter(stdouts...)

	stderrs := append(make([]io.Writer, 0, 2), &stderr)
	if sp.logStderr {
		stderrs = append(stderrs, printing.NewLinePrefixWriter(sp.log, errPrefix))
	}
	sp.cmd.Stderr = io.MultiWriter(stderrs...)

	sp.log.Logf("Running %q with args %q...", sp.cmd.Path, sp.cmd.Args[1:])
	err := sp.cmd.Run()
	if err != nil {
		sp.log.Logf("Command failed: %v", err)
	} else {
		sp.log.Logf("Command completed successfully")
	}

	return stdout.String(), stderr.String(), err
}

// Args returns the provided variadic args as a slice. This allows
// you to use Cmd like this:
//
//     run.Cmd("echo", run.Args("hello", "world"))
//
// The above is arguably slightly more readable than using a []string
// directly:
//
//     run.Cmd("echo", []string{"hello", "world"})
func Args(args ...string) []string {
	return args
}

// Option alters the way Cmd runs the provided command.
type Option func(*cmdinfo)

// Env causes Cmd to set *additional* environment variables for the command.
func Env(env ...string) Option {
	return func(s *cmdinfo) {
		if len(s.cmd.Env) == 0 {
			s.cmd.Env = append(os.Environ(), env...)
		} else {
			s.cmd.Env = append(s.cmd.Env, env...)
		}
	}
}

// Stdin causes Cmd to send the provided string to the command as stdin.
func Stdin(in string) Option {
	return func(s *cmdinfo) {
		s.cmd.Stdin = strings.NewReader(in)
	}
}

// Stdout causes Cmd to tee the Stdout of the process to the provided io.Writer.
func Stdout(out io.Writer) Option {
	return func(s *cmdinfo) {
		s.cmd.Stdout = out
	}
}

// Log changes where Cmd writes log-like information about running the command.
func Log(to io.Writer) Option {
	return func(s *cmdinfo) {
		s.log = printing.NewLogWriter(to)
	}
}

// SuppressStdout prevents Cmd from copying the command stdout (with each
// line prefixed with "  > ") to the writer configured with Log (os.Stdout
// by default).
func SuppressStdout() Option {
	return func(s *cmdinfo) {
		s.logStdout = false
	}
}

// SuppressStdout prevents Cmd from copying the command stderr (with each
// line prefixed with "  ! ") to the writer configured with Log (os.Stdout
// by default).
func SuppressStderr() Option {
	return func(s *cmdinfo) {
		s.logStderr = false
	}
}

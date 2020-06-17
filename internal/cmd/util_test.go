package cmd

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/subcommands"

	"github.com/stretchr/testify/require"
)

func Test_executeNoArgs(t *testing.T) {
	f := flag.NewFlagSet("foo", flag.ContinueOnError)
	err := f.Parse(nil)
	require.NoError(t, err)
	called := false
	exitStatus := executeNoArgs(f, func() error { called = true; return nil })
	require.True(t, called)
	require.Equal(t, subcommands.ExitSuccess, exitStatus)
}

func Test_executeNoArgs_argsProvided(t *testing.T) {
	f := flag.NewFlagSet("foo", flag.ContinueOnError)
	err := f.Parse([]string{"somearg"})
	require.NoError(t, err)
	called := false
	exitStatus := executeNoArgs(f, func() error { called = true; return nil })
	require.False(t, called)
	require.Equal(t, subcommands.ExitUsageError, exitStatus)
}

func Test_executeNoArgs_implReturnsError(t *testing.T) {
	f := flag.NewFlagSet("foo", flag.ContinueOnError)
	err := f.Parse(nil)
	require.NoError(t, err)
	called := false
	exitStatus := executeNoArgs(f, func() error { called = true; return errors.New("i'm broken") })
	require.True(t, called)
	require.Equal(t, subcommands.ExitFailure, exitStatus)
}

// pushd is a test utility that changes the current directory and returns a
// function (suitable for defer) that will change it back.
func pushd(t *testing.T, elem ...string) func() {
	prevDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(filepath.Join(elem...))
	require.NoError(t, err)
	return func() {
		err := os.Chdir(prevDir)
		require.NoError(t, err)
	}
}

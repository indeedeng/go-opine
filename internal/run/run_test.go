package run

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_Cmd(t *testing.T) {
	stdout, stderr, err := Cmd("echo", Args("hello", "world"))
	require.NoError(t, err)
	require.Equal(t, "hello world\n", stdout)
	require.Empty(t, stderr)
}

func Test_Cmd_error(t *testing.T) {
	stdout, stderr, err := Cmd("sh", Args("-c", "echo hello; >&2 echo 'world'; exit 1"))
	require.Error(t, err)
	require.Equal(t, "hello\n", stdout)
	require.Equal(t, "world\n", stderr)
}

func Test_Cmd_optionEnv(t *testing.T) {
	val := time.Now().Format(time.RFC3339Nano)
	stdout, stderr, err := Cmd("sh", Args("-c", "echo ${GO_OPINE_TEST_ENV}"), Env("GO_OPINE_TEST_ENV="+val))
	require.NoError(t, err)
	require.Equal(t, val+"\n", stdout)
	require.Empty(t, stderr)
}

func Test_Cmd_optionEnv_multipleTimes(t *testing.T) {
	val1 := time.Now().Format(time.RFC3339Nano)
	val2 := val1 + " (OVERRIDDEN)"
	val3 := val1 + " (ADDITIONAL)"
	stdout, stderr, err := Cmd(
		"sh",
		Args("-c", "echo ${GO_OPINE_ORIGINAL} ${GO_OPINE_OVERRIDDEN} ${GO_OPINE_TEST_ENV}"),
		Env("GO_OPINE_ORIGINAL="+val1, "GO_OPINE_OVERRIDDEN="+val1),
		Env("GO_OPINE_OVERRIDDEN="+val2, "GO_OPINE_TEST_ENV="+val3),
	)
	require.NoError(t, err)
	require.Equal(t, val1+" "+val2+" "+val3+"\n", stdout)
	require.Empty(t, stderr)
}

func Test_Cmd_optionStdin(t *testing.T) {
	val := time.Now().Format(time.RFC3339Nano)
	stdout, stderr, err := Cmd("cat", Args(), Stdin(val))
	require.NoError(t, err)
	require.Equal(t, val, stdout)
	require.Empty(t, stderr)
}

func Test_Cmd_optionLog(t *testing.T) {
	var log bytes.Buffer
	stdout, stderr, err := Cmd("sh", Args("-c", "echo hello; >&2 echo 'world'"), Log(&log))
	require.NoError(t, err)
	require.Equal(t, "hello\n", stdout)
	require.Equal(t, "world\n", stderr)
	require.Contains(t, log.String(), "  > hello\n")
	require.Contains(t, log.String(), "  ! world\n")
}

func Test_Cmd_optionSuppressStdout(t *testing.T) {
	var log bytes.Buffer
	stdout, stderr, err := Cmd("sh", Args("-c", "echo hello; >&2 echo 'world'"), Log(&log), SuppressStdout())
	require.NoError(t, err)
	require.Equal(t, "hello\n", stdout)
	require.Equal(t, "world\n", stderr)
	require.NotContains(t, log.String(), "  > hello\n")
	require.Contains(t, log.String(), "  ! world\n")
}

func Test_Cmd_optionSuppressStderr(t *testing.T) {
	var log bytes.Buffer
	stdout, stderr, err := Cmd("sh", Args("-c", "echo hello; >&2 echo 'world'"), Log(&log), SuppressStderr())
	require.NoError(t, err)
	require.Equal(t, "hello\n", stdout)
	require.Equal(t, "world\n", stderr)
	require.Contains(t, log.String(), "  > hello\n")
	require.NotContains(t, log.String(), "  ! world\n")
}

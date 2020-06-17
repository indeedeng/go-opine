package printing

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_LogWriter_Write(t *testing.T) {
	toWrite := []byte("test")
	var b bytes.Buffer
	tested := NewLogWriter(&b)
	n, err := tested.Write(toWrite)
	require.NoError(t, err)
	require.Equal(t, len(toWrite), n)
	require.Equal(t, "test", b.String())
}

func Test_LogWriter_Write_error(t *testing.T) {
	toWrite := []byte("test")
	tested := NewLogWriter(errorWriter{n: 2})
	n, err := tested.Write(toWrite)
	require.NoError(t, err)
	require.Equal(t, len(toWrite), n)
}

func Test_LogWriter_Log(t *testing.T) {
	var b bytes.Buffer
	tested := NewLogWriter(&b)
	tested.Log("a %s %s", "b", "c")
	require.Equal(t, "a b c\n", b.String())
}

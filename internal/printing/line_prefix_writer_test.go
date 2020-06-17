package printing

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_LinePrefixWriter_Write(t *testing.T) {
	toWrite := []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit.\nSed nisi lorem, cursus ac sagittis et, tincidunt sed mi.\nCurabitur vel nisi nec velit fermentum rhoncus.\n")
	var b bytes.Buffer
	tested := NewLinePrefixWriter(&b, "TEST ")
	n, err := tested.Write(toWrite)
	require.NoError(t, err)
	require.Equal(t, len(toWrite), n)
	require.Equal(t, "TEST Lorem ipsum dolor sit amet, consectetur adipiscing elit.\nTEST Sed nisi lorem, cursus ac sagittis et, tincidunt sed mi.\nTEST Curabitur vel nisi nec velit fermentum rhoncus.\n", b.String())
}

func Test_LinePrefixWriter_Write_noNewlineAtEnd(t *testing.T) {
	toWrite := []byte("Lorem ipsum\ndolor sit amet\nconsectetur adipiscing")
	var b bytes.Buffer
	tested := NewLinePrefixWriter(&b, "TEST ")
	n, err := tested.Write(toWrite)
	require.NoError(t, err)
	require.Equal(t, len(toWrite), n)
	require.Equal(t, "TEST Lorem ipsum\nTEST dolor sit amet\nTEST consectetur adipiscing", b.String())
}

func Test_LinePrefixWriter_Write_continueMidLine(t *testing.T) {
	toWrite1 := []byte("Lorem ipsum dolor ")
	toWrite2 := []byte("sit amet")
	var b bytes.Buffer
	tested := NewLinePrefixWriter(&b, "TEST ")
	n, err := tested.Write(toWrite1)
	require.NoError(t, err)
	require.Equal(t, len(toWrite1), n)
	require.Equal(t, "TEST Lorem ipsum dolor ", b.String())
	n, err = tested.Write(toWrite2)
	require.NoError(t, err)
	require.Equal(t, len(toWrite2), n)
	require.Equal(t, "TEST Lorem ipsum dolor sit amet", b.String())
}

func Test_LinePrefixWriter_Write_continueMidLineWithNewline(t *testing.T) {
	toWrite1 := []byte("Lorem ipsum dolor")
	toWrite2 := []byte("\nsit amet")
	var b bytes.Buffer
	tested := NewLinePrefixWriter(&b, "TEST ")
	n, err := tested.Write(toWrite1)
	require.NoError(t, err)
	require.Equal(t, len(toWrite1), n)
	require.Equal(t, "TEST Lorem ipsum dolor", b.String())
	n, err = tested.Write(toWrite2)
	require.NoError(t, err)
	require.Equal(t, len(toWrite2), n)
	require.Equal(t, "TEST Lorem ipsum dolor\nTEST sit amet", b.String())
}

func Test_LinePrefixWriter_Write_continueNextLine(t *testing.T) {
	toWrite1 := []byte("Lorem ipsum dolor\n")
	toWrite2 := []byte("sit amet")
	var b bytes.Buffer
	tested := NewLinePrefixWriter(&b, "TEST ")
	n, err := tested.Write(toWrite1)
	require.NoError(t, err)
	require.Equal(t, len(toWrite1), n)
	require.Equal(t, "TEST Lorem ipsum dolor\n", b.String())
	n, err = tested.Write(toWrite2)
	require.NoError(t, err)
	require.Equal(t, len(toWrite2), n)
	require.Equal(t, "TEST Lorem ipsum dolor\nTEST sit amet", b.String())
}

func Test_LinePrefixWriter_Write_errorInPrefix(t *testing.T) {
	toWrite1 := []byte("NEVER WRITTEN")
	tested := NewLinePrefixWriter(errorWriter{n: 5}, "12345|<- ERROR HERE")
	n, err := tested.Write(toWrite1)
	require.Error(t, err)
	require.Zero(t, n)
}

func Test_LinePrefixWriter_Write_errorAfterPrefix(t *testing.T) {
	toWrite1 := []byte("6789 |<- ERROR HERE")
	tested := NewLinePrefixWriter(errorWriter{n: 9}, "12345")
	n, err := tested.Write(toWrite1)
	require.Error(t, err)
	require.Equal(t, 4, n) // Prefix is not counted as written bytes.
}

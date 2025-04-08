package printing

import "errors"

var errorWriterErr = errors.New("i'm broken")

// errorWriter is a writer that always errors (for testing).
type errorWriter struct {
	n int
}

// Always returns n bytes written and errorWriterErr.
func (f errorWriter) Write([]byte) (int, error) {
	return f.n, errorWriterErr
}

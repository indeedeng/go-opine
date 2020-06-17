package printing

import (
	"bytes"
	"io"
)

// NewLinePrefixWriter creates and returns a new LinePrefixWriter.
func NewLinePrefixWriter(to io.Writer, prefix string) *LinePrefixWriter {
	return &LinePrefixWriter{
		to:     to,
		prefix: []byte(prefix),
	}
}

// NewLinePrefixWriter wraps an io.Writer and adds a prefix to every line.
type LinePrefixWriter struct {
	to                 io.Writer
	prefix             []byte
	suppressNextPrefix bool
}

var _ io.Writer = (*LinePrefixWriter)(nil)

// Write to the underlying io.Writer, prefixing each new line with the
// configured prefix. Returns the number of *input* bytes written (i.e.
// not including the prefix bytes written) and whether an error occurred.
// If no error occurred the number of bytes written will always be equal
// to the length of the input.
func (w *LinePrefixWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	n := 0
	rest := p
	for len(rest) > 0 {
		var line []byte

		if i := bytes.IndexByte(rest, '\n'); i >= 0 {
			line = rest[:i+1]
			rest = rest[i+1:]
		} else {
			line = rest
			rest = nil
		}

		var prefix []byte
		if !w.suppressNextPrefix {
			prefix = w.prefix
		}

		if cnt, err := w.to.Write(append(prefix, line...)); err != nil {
			if cnt < len(prefix) {
				cnt = 0
			} else {
				cnt -= len(prefix)
			}
			return n + cnt, err
		}
		n += len(line)
		w.suppressNextPrefix = false
	}

	w.suppressNextPrefix = !bytes.HasSuffix(p, []byte{'\n'})

	return n, nil
}

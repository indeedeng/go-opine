package printing

import (
	"fmt"
	"io"
	"sync"
)

func NewLogWriter(to io.Writer) *LogWriter {
	return &LogWriter{out: to}
}

// LogWriter wraps an io.Writer and ignores errors when writing. It also
// provides a Log method similar to log.Printf.
type LogWriter struct {
	out io.Writer
	mu  sync.Mutex
}

var _ io.Writer = (*LogWriter)(nil)

// Write to the underlying io.Writer. Errors are silently ignored. Always
// returns len(p) and a nil error.
func (lw *LogWriter) Write(p []byte) (n int, err error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	_, _ = lw.out.Write(p)
	return len(p), nil
}

// Log will write a log line to the underlying io.Writer. A newline is
// automatically appended.
func (lw *LogWriter) Log(format string, a ...interface{}) {
	_, _ = fmt.Fprintf(lw, format+"\n", a...)
}

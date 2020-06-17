package gotest

import (
	"bufio"
	"encoding/json"
	"io"
	"regexp"
	"time"
)

var workaroundGoIssue35169Regexp = regexp.MustCompile(`^FAIL\s+(\S+)\s+\[build failed\]$`)

// event is a test event printed by "go test -json". See
// "go doc test2json" for more details. This struct was
// copied directly from those docs.
type event struct {
	Time    time.Time // encodes as an RFC3339-format string
	Action  string
	Package string
	Test    string
	Elapsed float64 // seconds
	Output  string
}

// eventAccepter accepts events created by an eventStreamParser.
type eventAccepter interface {
	Accept(e event) error
}

// eventConverter converts a single line (without the newline)
// to events.
type eventConverter interface {
	Convert(line []byte) ([]event, error)
}

// jsonEventConverter converts JSON event lines (as printed by
// "go test -json") into singleton lists with the corresponding
// event.
type jsonEventConverter struct {
}

var _ eventConverter = jsonEventConverter{}

func (jsonEventConverter) Convert(line []byte) ([]event, error) {
	var e event
	if err := json.Unmarshal(line, &e); err != nil {
		return nil, err
	}
	return []event{e}, nil
}

// workaroundGoIssue35169EventConverter first calls the primary
// eventConverter and, if that fails, falls back to converting
// lines like "FAIL	example.com [build failed]" into events.
//
// This is to work around https://github.com/golang/go/issues/35169.
type workaroundGoIssue35169EventConverter struct {
	primary eventConverter
}

var _ eventConverter = (*workaroundGoIssue35169EventConverter)(nil)

func (w *workaroundGoIssue35169EventConverter) Convert(line []byte) ([]event, error) {
	events, err := w.primary.Convert(line)
	if err == nil {
		return events, nil
	}

	match := workaroundGoIssue35169Regexp.FindSubmatch(line)
	if match != nil {
		ts := time.Now()
		pkg := string(match[1])
		events := []event{
			{
				Time:    ts,
				Action:  "output",
				Package: pkg,
				Output:  string(line) + "\n", // bufio.Sanner removes the newline
			},
			{
				Time:    ts,
				Action:  "fail",
				Package: pkg,
				Elapsed: 0,
			},
		}
		return events, nil
	}

	return nil, err
}

// eventStreamParser reads "go test -json" output, converts
// each line to an event, and passes each event to the eventAccepter.
type eventStreamParser struct {
	to        eventAccepter
	converter eventConverter
}

func newEventStreamParser(to eventAccepter) *eventStreamParser {
	return &eventStreamParser{
		to:        to,
		converter: &workaroundGoIssue35169EventConverter{jsonEventConverter{}},
	}
}

// Parse "go test -json" output into events and pass them to the
// eventAccepter.
//
// If any line is not JSON, or if the eventAccepter returns an
// error then Parse will stop immediately and return the error.
func (esp *eventStreamParser) Parse(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		events, err := esp.converter.Convert(scanner.Bytes())
		if err != nil {
			return err
		}
		for _, e := range events {
			if err := esp.to.Accept(e); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

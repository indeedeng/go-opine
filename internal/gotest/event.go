package gotest

import (
	"bufio"
	"encoding/json"
	"io"
	"time"
)

// event is a test event printed by "go test -json". See
// "go doc test2json" for more details. This struct was
// copied directly from those docs.
type event struct {
	Time        time.Time // encodes as an RFC3339-format string
	Action      string
	Package     string
	Test        string
	Elapsed     float64 // seconds
	Output      string
	FailedBuild string
	ImportPath  string
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

// eventStreamParser reads "go test -json" output, converts
// each line to an event, and passes each event to the eventAccepter.
type eventStreamParser struct {
	to        eventAccepter
	converter eventConverter
}

func newEventStreamParser(to eventAccepter) *eventStreamParser {
	return &eventStreamParser{
		to:        to,
		converter: &jsonEventConverter{},
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

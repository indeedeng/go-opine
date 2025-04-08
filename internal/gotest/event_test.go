package gotest

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type eventAccepterFunc func(e event) error

func (f eventAccepterFunc) Accept(e event) error {
	return f(e)
}

func Test_eventStreamParser_Parse(t *testing.T) {
	// Taken from "go test -v -json" output.
	const event1JSON = `{"Time":"2019-09-26T13:27:17.563229183Z","Action":"output","Package":"oss.indeed.com/go/go-opine/internal/cmd","Test":"Test_testCmd_impl","Output":"--- PASS: Test_testCmd_impl (1.93s)\n"}`
	const event2JSON = `{"Time":"2019-09-26T13:27:17.56324465Z","Action":"pass","Package":"oss.indeed.com/go/go-opine/internal/cmd","Test":"Test_testCmd_impl","Elapsed":1.93}`

	var events []event
	tested := newEventStreamParser(eventAccepterFunc(func(e event) error { events = append(events, e); return nil }))
	err := tested.Parse(strings.NewReader(event1JSON + "\n" + event2JSON + "\n"))
	require.NoError(t, err)
	require.Equal(
		t,
		[]event{
			{
				Time:    time.Date(2019, 9, 26, 13, 27, 17, 563229183, time.UTC),
				Action:  "output",
				Package: "oss.indeed.com/go/go-opine/internal/cmd",
				Test:    "Test_testCmd_impl",
				Output:  "--- PASS: Test_testCmd_impl (1.93s)\n",
			},
			{
				Time:    time.Date(2019, 9, 26, 13, 27, 17, 563244650, time.UTC),
				Action:  "pass",
				Package: "oss.indeed.com/go/go-opine/internal/cmd",
				Test:    "Test_testCmd_impl",
				Elapsed: 1.93,
			},
		},
		events,
	)
}

func Test_eventStreamParser_Parse_noTrailingNewline(t *testing.T) {
	// Taken from "go test -v -json" output.
	const eventJSON = `{"Time":"2019-09-26T13:27:17.563229183Z","Action":"output","Package":"oss.indeed.com/go/go-opine/internal/cmd","Test":"Test_testCmd_impl","Output":"--- PASS: Test_testCmd_impl (1.93s)\n"}`

	var events []event
	tested := newEventStreamParser(eventAccepterFunc(func(e event) error { events = append(events, e); return nil }))
	err := tested.Parse(strings.NewReader(eventJSON))
	require.NoError(t, err)
	require.Equal(
		t,
		[]event{
			{
				Time:    time.Date(2019, 9, 26, 13, 27, 17, 563229183, time.UTC),
				Action:  "output",
				Package: "oss.indeed.com/go/go-opine/internal/cmd",
				Test:    "Test_testCmd_impl",
				Output:  "--- PASS: Test_testCmd_impl (1.93s)\n",
			},
		},
		events,
	)
}

func Test_eventStreamParser_Parse_emptyEvent(t *testing.T) {
	var events []event
	tested := newEventStreamParser(eventAccepterFunc(func(e event) error { events = append(events, e); return nil }))
	err := tested.Parse(strings.NewReader("{}\n"))
	require.NoError(t, err)
	require.Equal(t, []event{{}}, events)
}

func Test_eventStreamParser_Parse_unmarshalError(t *testing.T) {
	const notJSON = "=== RUN   Test_eventStreamParser_Parse"
	var events []event
	tested := newEventStreamParser(eventAccepterFunc(func(e event) error { events = append(events, e); return nil }))
	err := tested.Parse(strings.NewReader(notJSON + "\n"))
	require.Error(t, err)
	require.Empty(t, events)
}

func Test_eventStreamParser_Parse_acceptError(t *testing.T) {
	expectedErr := errors.New("failed to parse")
	tested := newEventStreamParser(eventAccepterFunc(func(event) error { return expectedErr }))
	err := tested.Parse(strings.NewReader("{}\n"))
	require.Equal(t, expectedErr, err)
}

func Test_eventStreamParser_Parse_workaroundGoIssue35169(t *testing.T) {
	// Taken from "go test -v -json" output.
	const eventNotJSON = `FAIL	example.com [build failed]`

	var events []event
	tested := newEventStreamParser(
		eventAccepterFunc(func(e event) error {
			require.NotZero(t, e.Time)
			e.Time = time.Time{}
			events = append(events, e)
			return nil
		}),
	)
	err := tested.Parse(strings.NewReader(eventNotJSON))
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(
		t,
		[]event{
			{
				Action:  "output",
				Package: "example.com",
				Output:  eventNotJSON + "\n",
			},
			{
				Action:  "fail",
				Package: "example.com",
			},
		},
		events,
	)
}

package gotest

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type eventAccepterFunc func(e event) error

func (f eventAccepterFunc) Accept(e event) error {
	return f(e)
}

func Test_eventStreamParser(t *testing.T) {
	type testcase struct {
		Description  string
		GoTestOutput string

		ExpectedEvent event
		ExpectedError bool
	}
	testcases := []testcase{
		{
			Description:  "event parser should parse an test output event",
			GoTestOutput: `{"Time":"2019-09-26T13:27:17.563229183Z","Action":"output","Package":"oss.indeed.com/go/go-opine/internal/cmd","Test":"Test_testCmd_impl","Output":"--- PASS: Test_testCmd_impl (1.93s)\n"}`,

			ExpectedEvent: event{
				Time:    time.Date(2019, 9, 26, 13, 27, 17, 563229183, time.UTC),
				Action:  "output",
				Package: "oss.indeed.com/go/go-opine/internal/cmd",
				Test:    "Test_testCmd_impl",
				Output:  "--- PASS: Test_testCmd_impl (1.93s)\n",
			},
		},
		{
			Description:  "event parser should parse an test pass event",
			GoTestOutput: `{"Time":"2019-09-26T13:27:17.56324465Z","Action":"pass","Package":"oss.indeed.com/go/go-opine/internal/cmd","Test":"Test_testCmd_impl","Elapsed":1.93}`,

			ExpectedEvent: event{
				Time:    time.Date(2019, 9, 26, 13, 27, 17, 563244650, time.UTC),
				Action:  "pass",
				Package: "oss.indeed.com/go/go-opine/internal/cmd",
				Test:    "Test_testCmd_impl",
				Elapsed: 1.93,
			},
		},
		{
			Description:  "event parser should parse an empty event",
			GoTestOutput: `{}`,

			ExpectedEvent: event{},
		},
		{
			Description:  "event parser should parse an build-output event",
			GoTestOutput: `{"ImportPath":"oss.indeed.com/go/go-opine/internal/gotest","Action":"build-output","Output":"# oss.indeed.com/go/go-opine/internal/gotest\n"}`,

			ExpectedEvent: event{
				Action:     "build-output",
				Output:     "# oss.indeed.com/go/go-opine/internal/gotest\n",
				ImportPath: "oss.indeed.com/go/go-opine/internal/gotest",
			},
		},
		{
			Description:  "event parser should parse an build-fail event",
			GoTestOutput: `{"ImportPath":"oss.indeed.com/go/go-opine/internal/gotest","Action":"build-fail"}`,

			ExpectedEvent: event{
				Action:     "build-fail",
				ImportPath: "oss.indeed.com/go/go-opine/internal/gotest",
			},
		},
		{
			Description:  "event parser should parse events with build-output warnings",
			GoTestOutput: `{"ImportPath":"oss.indeed.com/test/datadogquery/resourcemetrics.test","Action":"build-output","Output":"# oss.indeed.com/test/datadogquery/resourcemetrics.test\nld: warning: '/private/var/folders/2y/4sct57296733gvqgbczv_x5w0000gn/T/go-link-1858153711/000055.o' has malformed LC_DYSYMTAB, expected 98 undefined symbols to start at index 1626, found 95 undefined symbols starting at index 1626\n"}`,
			ExpectedEvent: event{
				Action:     "build-output",
				ImportPath: "oss.indeed.com/test/datadogquery/resourcemetrics.test",
				Output:     "# oss.indeed.com/test/datadogquery/resourcemetrics.test\nld: warning: '/private/var/folders/2y/4sct57296733gvqgbczv_x5w0000gn/T/go-link-1858153711/000055.o' has malformed LC_DYSYMTAB, expected 98 undefined symbols to start at index 1626, found 95 undefined symbols starting at index 1626\n",
			},
		},
		{
			Description:  "event parser should fail to parse text output",
			GoTestOutput: `=== RUN   Test_eventStreamParser`,

			ExpectedError: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Description, func(t *testing.T) {
			var observedEvent event
			parser := newEventStreamParser(eventAccepterFunc(func(e event) error {
				observedEvent = e
				return nil
			}))
			err := parser.Parse(strings.NewReader(tc.GoTestOutput))
			if tc.ExpectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.ExpectedEvent, observedEvent)
			}
		})
	}
}

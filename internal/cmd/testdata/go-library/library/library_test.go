package library

import (
	"os"
	"testing"
)

func Test_Library(t *testing.T) {
	Library()
	if os.Getenv("LIBRARY_FAIL_UNIT_TESTS") == "1" {
		t.Fail()
	}
}

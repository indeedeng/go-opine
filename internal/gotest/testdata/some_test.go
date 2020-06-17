package testdata

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Some_test(t *testing.T) {
	if os.Getenv("GOTEST_FAIL") == "1" {
		require.Fail(t, "FAIL")
	}
	require.Equal(t, 42, Some{}.test())
}

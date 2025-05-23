package coverage

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"golang.org/x/tools/cover"
)

func Test_Load(t *testing.T) {
	inPath := filepath.Join("testdata", "cover.out")
	result, err := Load(inPath)
	require.NoError(t, err)

	// Check that the generated file was removed and the non-generated
	// file was not.
	files := make([]string, len(result.profiles))
	for i, profile := range result.profiles {
		files[i] = profile.FileName
	}
	require.Contains(t, files, "oss.indeed.com/go/go-opine/internal/coverage/testdata/not_generated.go")
	require.NotContains(t, files, "oss.indeed.com/go/go-opine/internal/coverage/testdata/generated.go")

	// Check that the modPaths is populated and correct.
	expectedModPath, err := filepath.Abs("./testdata")
	require.NoError(t, err)
	require.Equal(
		t,
		map[string]string{"oss.indeed.com/go/go-opine/internal/coverage/testdata": expectedModPath},
		result.modPaths,
	)
}

func Test_CoverProfile(t *testing.T) {
	inPath := filepath.Join("testdata", "cover.out")
	profiles, err := cover.ParseProfiles(inPath)
	require.NoError(t, err)

	outDir, err := os.MkdirTemp("", "go-opine-coverage-test.")
	require.NoError(t, err)
	defer os.RemoveAll(outDir)
	outPath := filepath.Join(outDir, "cover.out")

	cov := &Coverage{profiles: profiles}
	err = cov.CoverProfile(outPath)
	require.NoError(t, err)

	inBytes, err := os.ReadFile(inPath)
	require.NoError(t, err)
	outBytes, err := os.ReadFile(outPath)
	require.NoError(t, err)
	require.Equal(t, string(inBytes), string(outBytes))
}

func Test_XML(t *testing.T) {
	inPath := filepath.Join("testdata", "cover.out")
	profiles, err := cover.ParseProfiles(inPath)
	require.NoError(t, err)

	outDir, err := os.MkdirTemp("", "go-opine-coverage-test.")
	require.NoError(t, err)
	defer os.RemoveAll(outDir)
	outPath := filepath.Join(outDir, "lang-go-cobertura.xml")

	testdata, err := filepath.Abs("./testdata")
	require.NoError(t, err)
	cov := &Coverage{
		profiles: profiles,
		modPaths: map[string]string{"oss.indeed.com/go/go-opine/internal/coverage/testdata": testdata},
	}
	err = cov.XML(outPath)
	require.NoError(t, err)
	outBytes, err := os.ReadFile(outPath)
	require.NoError(t, err)
	out := string(outBytes)
	require.Contains(t, out, "\"testdata/not_generated.go\"")
	require.Contains(t, out, "\"testdata/generated.go\"") // included because we did not use Load
}

func Test_Ratio_50percent(t *testing.T) {
	inPath := filepath.Join("testdata", "cover.out")
	profiles, err := cover.ParseProfiles(inPath) // 50% because generated.go is not excluded
	require.NoError(t, err)
	cov := &Coverage{profiles: profiles}
	ratio := cov.Ratio()
	require.Equal(t, 0.5, ratio)
}

func Test_Ratio_100percent(t *testing.T) {
	inPath := filepath.Join("testdata", "cover.out")
	cov, err := Load(inPath) // 100% because generated.go is excluded
	require.NoError(t, err)
	ratio := cov.Ratio()
	require.Equal(t, 1.0, ratio)
}
func Test_isGeneratedReader_veryLongLine(t *testing.T) {
	veryLongLine := "package foo\n\n" +
		"// " + strings.Repeat("a", bufio.MaxScanTokenSize+1) +
		"\n\n// Code generated ... DO NOT EDIT.\n"
	res, err := isGeneratedReader(strings.NewReader(veryLongLine))
	require.NoError(t, err)
	require.True(t, res)
}

func Test_isGeneratedReader_runeReaderErr(t *testing.T) {
	_, err := isGeneratedReader(notEOFErrRuneReader{})
	require.Error(t, err)
}

type notEOFErrRuneReader struct{}

func (notEOFErrRuneReader) ReadRune() (rune, int, error) {
	return 0, 0, errors.New("not EOF")
}

// Package coverage is for writing coverage reports.
package coverage

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/cover"

	"oss.indeed.com/go/go-opine/internal/run"
)

// Generate testdata/cover.out by running the ./testdata tests.
//go:generate go test -v -race -coverprofile=testdata/cover.out -covermode=atomic ./testdata

var generatedFileRegexp = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

type Coverage struct {
	profiles []*cover.Profile
	modPaths map[string]string
}

// Load a Go coverprofile file. Generated files are excluded from the result.
func Load(inPath string) (*Coverage, error) {
	profiles, err := cover.ParseProfiles(inPath)
	if err != nil {
		return nil, err
	}
	paths, err := findModPaths(profiles)
	if err != nil {
		return nil, err
	}
	profiles, err = profilesWithoutGenerated(profiles, paths)
	if err != nil {
		return nil, err
	}
	return &Coverage{profiles: profiles, modPaths: paths}, nil
}

// CoverProfile writes the coverage to a file in the Go "coverprofile" format.
func (cov *Coverage) CoverProfile(outPath string) error {
	f, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	err = writeProfiles(cov.profiles, f)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	return err
}

// XML writes the coverage to a file in the Cobertura-style XML format.
func (cov *Coverage) XML(outPath string) error {
	// Work around an issue with gocover-cobertura and Jenkins where the files
	// cannot be found.
	profilesWithRealPaths := make([]*cover.Profile, len(cov.profiles))
	for i, profile := range cov.profiles {
		fileLoc, err := findFileRel(profile.FileName, cov.modPaths)
		if err != nil {
			return err
		}
		profileWithRealPath := *profile
		profileWithRealPath.FileName = fileLoc
		profilesWithRealPaths[i] = &profileWithRealPath
	}
	var modifiedGoCov strings.Builder
	if err := writeProfiles(profilesWithRealPaths, &modifiedGoCov); err != nil {
		return err
	}
	xmlCovOut, _, err := run.Cmd(
		"go",
		run.Args("run", "github.com/t-yuki/gocover-cobertura"),
		run.Stdin(modifiedGoCov.String()),
		run.SuppressStdout(),
	)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(outPath, []byte(xmlCovOut), 0666) //nolint:gosec
}

// Ratio returns the ratio of covered statements over all statements. The
// value returned will always be between 0 and 1. If there are no statements
// then 1 is returned.
func (cov *Coverage) Ratio() float64 {
	statementCnt := 0
	statementHit := 0
	for _, profile := range cov.profiles {
		for _, block := range profile.Blocks {
			statementCnt += block.NumStmt
			if block.Count > 0 {
				statementHit += block.NumStmt
			}
		}
	}
	if statementCnt == 0 {
		return 1
	}
	return float64(statementHit) / float64(statementCnt)
}

// isGenerated checks if the provided file was generated or not. The file
// path should be a real filesystem path, not a Go module path (e.g.
// "/home/you/github.com/proj/main.go", not "github.com/proj/main.go").
func isGenerated(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close() // ignore close error (we are not writing)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if generatedFileRegexp.MatchString(scanner.Text()) {
			return true, nil
		}
	}
	return false, scanner.Err()
}

// profilesWithoutGenerated returns a new slice of profiles with all
// generated files removed. The provided modPaths must be a map from each
// known Go module to the absolute filesystem path of the module directory,
// as returned by findModPaths.
func profilesWithoutGenerated(profiles []*cover.Profile, modPaths map[string]string) ([]*cover.Profile, error) {
	res := make([]*cover.Profile, 0, len(profiles))
	for _, profile := range profiles {
		filePath, err := findFile(profile.FileName, modPaths)
		if err != nil {
			return nil, err
		}
		if gen, err := isGenerated(filePath); err != nil {
			return nil, err
		} else if !gen {
			res = append(res, profile)
		}
	}
	return res, nil
}

// findFile finds the absolute filesystem path of a file name specified
// relative to a Go module. The provided modPaths must be a map from each
// known Go module to the absolute filesystem path of the module directory,
// as returned by findModPaths.
//
// If the module of the provided file name is not in the paths an error is
// returned.
func findFile(fileName string, modPaths map[string]string) (string, error) {
	pkg := path.Dir(fileName)
	dir, ok := modPaths[pkg]
	if !ok {
		return "", fmt.Errorf("could not determine the filesystem path of %q", fileName)
	}
	return filepath.Join(dir, path.Base(fileName)), nil
}

// findFileRel finds the relative (to the current working directory)
// filesystem path of a file name specified relative to a Go module.
//
// See findFile for more information.
func findFileRel(fileName string, modPaths map[string]string) (string, error) {
	filePath, err := findFile(fileName, modPaths)
	if err != nil {
		return "", err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Rel(cwd, filePath)
}

// writeProfiles writes cover profiles to the provided io.Writer in the Go
// "coverprofile" format.
func writeProfiles(profiles []*cover.Profile, w io.Writer) error {
	if len(profiles) == 0 {
		return nil
	}

	if _, err := io.WriteString(w, "mode: "+profiles[0].Mode+"\n"); err != nil {
		return err
	}

	for _, profile := range profiles {
		for _, block := range profile.Blocks {
			_, err := fmt.Fprintf(
				w, "%s:%d.%d,%d.%d %d %d\n", profile.FileName,
				block.StartLine, block.StartCol, block.EndLine, block.EndCol,
				block.NumStmt, block.Count,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// findModPaths returns a map from module name to the filesystem path of
// the module directory. The filesystem paths are absolute and do not include
// a trailing "/".
func findModPaths(profiles []*cover.Profile) (map[string]string, error) {
	// Use %q to ensure the ImportPath and Dir boundaries can be easily
	// distinguished. This also protects us against file names containing
	// new line characters... you know you want to make them.
	mods := modsInProfiles(profiles)
	args := append(
		[]string{"list", "-f", `{{ .ImportPath | printf "%q" }} {{ .Dir | printf "%q" }}`},
		mods...,
	)
	stdout, stderr, err := run.Cmd("go", args, run.Log(ioutil.Discard))
	if err != nil {
		return nil, fmt.Errorf("error running go list [stdout=%q stderr=%q]: %w", stdout, stderr, err)
	}

	modPaths := make(map[string]string, len(mods))
	for _, line := range strings.Split(stdout, "\n") {
		if line == "" {
			continue
		}
		var (
			mod string
			dir string
		)
		_, err := fmt.Sscanf(line, "%q %q", &mod, &dir)
		if err != nil {
			return nil, fmt.Errorf("got unexpected line %q from go list: %w", line, err)
		}
		modPaths[mod] = dir
	}
	return modPaths, nil
}

// modsInProfiles returns a list of modules in the provided cover
// profiles. Duplicate modules will show up only once in the result.
func modsInProfiles(profiles []*cover.Profile) []string {
	mods := make([]string, 0)
	seen := make(map[string]bool)
	for _, profile := range profiles {
		mod, _ := filepath.Split(profile.FileName)
		if !seen[mod] {
			mods = append(mods, mod)
			seen[mod] = true
		}
	}
	return mods
}

//+build tools

// Package tools declares the tool dependencies required by go-opine.
//
// This approach is described in
// https://github.com/go-modules-by-example/index/tree/23a56e1095864bf596f2ce3aec296ecc89ab71b9/010_tools
// and https://github.com/golang/go/issues/25922#issuecomment-451123151.
package tools

import (
	_ "github.com/jstemmer/go-junit-report"

	_ "github.com/t-yuki/gocover-cobertura"
)

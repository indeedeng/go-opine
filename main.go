package main

import (
	"context"
	"flag"
	"os"

	"github.com/google/subcommands"

	"oss.indeed.com/go/go-opine/internal/cmd"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(cmd.TestCmd(), "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}

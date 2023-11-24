// cli is a command line interface to interact with cheops without using HTTP calls.
// It allows running subcommands to do specific stuff
//
// Available commands:
// - exec
// - show
//
// See the relevant files for more information

package main

import (
	"github.com/alecthomas/kong"
)

type CLI struct {
	Exec ExecCmd `cmd:"" help:"Run a command for a given resource"`
	Show ShowCmd `cmd:"" help:"Show resource listed by id, in all places it is supposed to be"`
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli, kong.UsageOnError())
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

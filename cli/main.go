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

package main

import (
	"github.com/alecthomas/kong"
)

type CLI struct {
	Exec ExecCmd `cmd:"" help:"Run a command for a given resource"`
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli, kong.UsageOnError())
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

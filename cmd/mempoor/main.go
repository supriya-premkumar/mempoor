package main

import (
	"context"
	"flag"
	"os"

	"mempoor/pkg/cmd"

	"github.com/google/subcommands"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(&cmd.NodeArgs{}, "")
	subcommands.Register(&cmd.TxArgs{}, "")
	subcommands.Register(&cmd.BlockArgs{}, "")

	flag.Parse()
	os.Exit(int(subcommands.Execute(context.Background())))

}

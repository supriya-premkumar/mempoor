package cmd

import (
	"context"
	"flag"
	"fmt"
	"mempoor/pkg/mempoor"
	"os"

	"github.com/google/subcommands"
)

type NodeArgs struct {
	listenAddr string
}

func (*NodeArgs) Name() string { return "start" }

func (*NodeArgs) Synopsis() string { return "starts a mempoor node" }

func (*NodeArgs) Usage() string {
	return `start [--flags]

Starts the mempoor node, which runs:

  • Mempool (pending transaction storage)
  • Block builder (produces finalized blocks)
  • RPC server  (accepts CLI commands)

Examples:
    mempoor start --listen 127.0.0.1:8080
`
}

func (args *NodeArgs) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&args.listenAddr, "listen", "127.0.0.1:8080", "address for the node to listen on")
}

func (args *NodeArgs) Execute(ctx context.Context, flagSet *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := mempoor.StartNode(ctx, args.listenAddr); err != nil {
		fmt.Fprintf(os.Stderr, "node error: %v\n", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}

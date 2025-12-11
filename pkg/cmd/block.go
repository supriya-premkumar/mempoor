package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

type BlockArgs struct {
	NodeAddr string
}

func (*BlockArgs) Name() string     { return "block" }
func (*BlockArgs) Synopsis() string { return "block-related commands" }
func (*BlockArgs) Usage() string {
	return `block <command> [--flags]

Block (chain) commands.

Blocks represent the FINALIZED output of the block builder. Once a
transaction is included in a block, it is removed from the mempool.
"mempoor block list" shows the chain history produced so far.

Commands:
    list        List all produced blocks (chain view)
    get         Get a specific block by height

Examples:
    # View all produced blocks (finalized chain view)
    mempoor block list

    # View a specific block
    mempoor block get --height 0
`
}

func (b *BlockArgs) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&b.NodeAddr, "addr", "localhost:8080", "address of running mempoor node")
}

func (b *BlockArgs) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() == 0 {
		fmt.Println(b.Usage())
		return subcommands.ExitUsageError
	}

	switch f.Arg(0) {
	case "list":
		return b.list(ctx)
	case "get":
		return b.get(ctx, f.Args()[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown block command: %s\n", f.Arg(0))
		return subcommands.ExitUsageError
	}
}

func (b *BlockArgs) list(ctx context.Context) subcommands.ExitStatus {
	params := map[string]interface{}{}

	var result struct {
		Blocks json.RawMessage `json:"blocks"`
	}

	if err := callRPC(b.NodeAddr, "block.list", params, &result); err != nil {
		fmt.Println("error:", err)
		return subcommands.ExitFailure
	}

	fmt.Println(string(result.Blocks))
	return subcommands.ExitSuccess
}

func (b *BlockArgs) get(ctx context.Context, args []string) subcommands.ExitStatus {
	fs := flag.NewFlagSet("block get", flag.ExitOnError)

	var height uint64
	fs.Uint64Var(&height, "height", 0, "block height")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitUsageError
	}

	params := map[string]interface{}{
		"height": height,
	}

	var result struct {
		Block json.RawMessage `json:"block"`
	}

	if err := callRPC(b.NodeAddr, "block.get", params, &result); err != nil {
		fmt.Println("error:", err)
		return subcommands.ExitFailure
	}

	fmt.Println(string(result.Block))
	return subcommands.ExitSuccess
}

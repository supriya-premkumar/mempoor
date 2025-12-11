package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/google/subcommands"
)

type TxArgs struct {
	NodeAddr string
}

func (*TxArgs) Name() string     { return "tx" }
func (*TxArgs) Synopsis() string { return "transaction operations: add, update, remove, list" }
func (*TxArgs) Usage() string {
	return `tx <command> [--flags]

Transaction (mempool) commands.

The mempool contains pending transactions that have NOT yet been included
in a block. "mempoor tx list" shows these transactions in PRIORITY ORDER,
i.e., the order in which the next block would include them.

Commands:
    add        Add a new transaction to the mempool
    update     Update the fee of an existing transaction
    remove     Remove a transaction from the mempool
    list       List current mempool transactions (priority-ordered)

Examples:
    # Add a transaction (pending in mempool)
    mempoor tx add --sender alice --recipient bob --fee 10 --gas 500

    # View pending transactions (mempool view)
    mempoor tx list

    # Update fee (RBF-like behavior)
    mempoor tx update --id <txid> --fee 100

    # Remove a pending tx
    mempoor tx remove --id <txid>
`
}

func (t *TxArgs) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&t.NodeAddr, "addr", "localhost:8080", "address of running mempoor node")
}

func (t *TxArgs) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() == 0 {
		fmt.Println(t.Usage())
		return subcommands.ExitUsageError
	}

	switch f.Arg(0) {
	case "add":
		return t.add(ctx, f.Args()[1:])
	case "update":
		return t.update(ctx, f.Args()[1:])
	case "remove":
		return t.remove(ctx, f.Args()[1:])
	case "list":
		return t.list(ctx)
	default:
		fmt.Fprintf(os.Stderr, "unknown tx command: %s\n", f.Arg(0))
		return subcommands.ExitUsageError
	}
}

func (t *TxArgs) add(ctx context.Context, args []string) subcommands.ExitStatus {
	fs := flag.NewFlagSet("tx add", flag.ExitOnError)

	var sender, recipient, payload string
	var fee, gas uint64

	fs.StringVar(&sender, "sender", "", "sender address")
	fs.StringVar(&recipient, "recipient", "", "recipient address")
	fs.StringVar(&payload, "payload", "", "payload")
	fs.Uint64Var(&fee, "fee", 0, "transaction fee")
	fs.Uint64Var(&gas, "gas", 0, "gas limit for transaction")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitUsageError
	}

	params := map[string]interface{}{
		"sender":    sender,
		"recipient": recipient,
		"payload":   payload,
		"fee":       fee,
		"gas":       gas,
	}

	var result struct {
		TxID string `json:"txID"`
	}

	if err := callRPC(t.NodeAddr, "tx.add", params, &result); err != nil {
		fmt.Println("error:", err)
		return subcommands.ExitFailure
	}

	fmt.Println("tx added:", result.TxID)
	return subcommands.ExitSuccess
}

func (t *TxArgs) update(ctx context.Context, args []string) subcommands.ExitStatus {
	fs := flag.NewFlagSet("tx update", flag.ExitOnError)

	var id string
	var fee uint64

	fs.StringVar(&id, "id", "", "transaction ID")
	fs.Uint64Var(&fee, "fee", 0, "new fee")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitUsageError
	}

	params := map[string]interface{}{
		"id":  id,
		"fee": fee,
	}

	var ok struct {
		OK bool `json:"ok"`
	}

	if err := callRPC(t.NodeAddr, "tx.update", params, &ok); err != nil {
		fmt.Println("error:", err)
		return subcommands.ExitFailure
	}

	fmt.Println("tx updated")
	return subcommands.ExitSuccess
}

func (t *TxArgs) remove(ctx context.Context, args []string) subcommands.ExitStatus {
	fs := flag.NewFlagSet("tx remove", flag.ExitOnError)

	var id string
	fs.StringVar(&id, "id", "", "transaction ID")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitUsageError
	}

	params := map[string]interface{}{"id": id}

	var ok struct {
		OK bool `json:"ok"`
	}

	if err := callRPC(t.NodeAddr, "tx.remove", params, &ok); err != nil {
		fmt.Println("error:", err)
		return subcommands.ExitFailure
	}

	fmt.Println("tx removed")
	return subcommands.ExitSuccess
}

func (t *TxArgs) list(ctx context.Context) subcommands.ExitStatus {
	params := map[string]interface{}{}

	var result struct {
		Transactions json.RawMessage `json:"transactions"`
	}

	if err := callRPC(t.NodeAddr, "tx.list", params, &result); err != nil {
		fmt.Println("error:", err)
		return subcommands.ExitFailure
	}

	fmt.Println(string(result.Transactions))
	return subcommands.ExitSuccess
}

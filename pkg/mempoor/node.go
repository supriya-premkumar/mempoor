package mempoor

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Node contains the mempool, block builder, chain history, and config.
// RPC handlers also live as methods on this struct.
type Node struct {
	mempool Mempool
	builder *BlockBuilder

	blocksMu sync.RWMutex
	blocks   []*Block

	cfg NodeConfig
}

// NewNode creates a fully initialized Node with mempool + builder.
func NewNode(cfg NodeConfig) *Node {
	mp := NewMempool()
	builder := NewBlockBuilder(mp, BlockBuilderConfig{
		GasLimit:      cfg.GasLimit,
		MaxTxPerBlock: cfg.MaxTxPerBlock,
		MinFee:        cfg.MinFee,
	})

	return &Node{
		mempool: mp,
		builder: builder,
		blocks:  make([]*Block, 0),
		cfg:     cfg,
	}
}

// StartNode is the public entrypoint called from CLI (NodeArgs.Execute).
// It sets up the node, HTTP server, and block production loop.
// All lifecycle control is driven by ctx.
func StartNode(ctx context.Context, listenAddr string) error {
	cfg := NodeConfig{
		ListenAddr:    listenAddr,
		BlockInterval: 2 * time.Second,
		GasLimit:      1_000_000,
		MaxTxPerBlock: 1000,
		MinFee:        0,
	}

	node := NewNode(cfg)
	return node.run(ctx)
}

func (n *Node) run(ctx context.Context) error {
	fmt.Printf("🚀 started mempoor node on %s\n", n.cfg.ListenAddr)

	// ---- Start HTTP server ----
	mux := http.NewServeMux()
	mux.HandleFunc("/rpc", n.handleRPC)

	server := &http.Server{
		Addr:    n.cfg.ListenAddr,
		Handler: mux,
	}

	errCh := make(chan error, 2)

	// HTTP server goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http server error: %w", err)
		}
	}()

	// ---- Start block production loop ----
	go func() {
		errCh <- n.runBlockLoop(ctx)
	}()

	// ---- Shutdown on ctx cancel ----
	select {
	case <-ctx.Done():
		_ = server.Shutdown(context.Background())
		fmt.Println("mempoor node shutting down:", ctx.Err())
		return nil

	case err := <-errCh:
		_ = server.Shutdown(context.Background())
		return err
	}
}

// runBlockLoop executes the block builder loop in a ticker.
// Only produces blocks when mempool has eligible txs.
func (n *Node) runBlockLoop(ctx context.Context) error {
	var (
		height   uint64
		prevHash [32]byte
	)

	ticker := time.NewTicker(n.cfg.BlockInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			now := time.Now().UTC()
			block, err := n.builder.BuildBlock(prevHash, height, now)
			if err == ErrEmptyBlock {
				continue // No block this round (mempool empty or txs below MinFee)
			}
			if err != nil {
				fmt.Printf("block build error at height %d: %v\n", height, err)
				continue
			}

			// Store block in memory
			n.blocksMu.Lock()
			n.blocks = append(n.blocks, block)
			n.blocksMu.Unlock()

			// Print summary
			printBlock(block)

			// Advance chain tip
			prevHash = block.Hash()
			height++
		}
	}
}

// ---- Helper for stdout block output ----

func printBlock(b *Block) {
	fmt.Printf(
		"BLOCK height=%d txs=%d gasUsed=%d hash=%x prevHash=%x time=%s\n",
		b.Header.Height,
		b.Header.TxCount,
		b.Header.GasUsed,
		b.Hash(),
		b.Header.PrevHash,
		b.Header.Timestamp.Format(time.RFC3339Nano),
	)
}

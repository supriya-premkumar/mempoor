package mempoor

import (
	"time"
)

// NewBlockBuilder constructs a builder with the given mempool and config.
func NewBlockBuilder(mp Mempool, cfg BlockBuilderConfig) *BlockBuilder {
	return &BlockBuilder{
		mp:  mp,
		cfg: cfg,
	}
}

// BuildBlock selects transactions under the configured constraints and
// constructs a block. If no transactions are available, ErrEmptyBlock is returned.
//
// prevHash: block hash of previous block in chain
// height: height of new block
// now: block timestamp (supplied by caller for determinism & testability)
func (b *BlockBuilder) BuildBlock(prevHash [32]byte, height uint64, now time.Time) (*Block, error) {
	// Build constraints for one block.
	constraints := BlockConstraints{
		GasLimit: b.cfg.GasLimit,
		MaxTx:    b.cfg.MaxTxPerBlock,
		MinFee:   b.cfg.MinFee,
	}

	// Ask mempool for the best transactions.
	selection := b.mp.SelectTransactions(constraints)

	if len(selection.Transactions) == 0 {
		return nil, ErrEmptyBlock
	}

	// Construct header with fields we have agreed upon.
	header := BlockHeader{
		Height:    height,
		PrevHash:  prevHash,
		Timestamp: now,
		TxCount:   len(selection.Transactions),
		GasUsed:   selection.GasUsed, // trust mempool per Q3
	}

	block := &Block{
		Header:       header,
		Transactions: selection.Transactions,
	}

	return block, nil
}

/*
PERFORMANCE NOTES:

1. Mempool.SelectTransactions already ensures a correct priority ordering,
   gas limit enforcement, and low-fee purging. BuildBlock does not duplicate
   that logic by design—this keeps the builder extremely lightweight and keeps
   the hot path in one place (the mempool).

2. Assembly is O(txCount) to build the header and return the block. This is
   negligible for realistic MaxTxPerBlock values unless extremely large blocks
   are used.

3. No hashing occurs inside BuildBlock; hashing is deferred to block.Hash().
   Many high-throughput chains separate block assembly from hashing for exactly
   this reason.

4. Additional block metadata (e.g. Merkle roots, signatures) can be added
   without modifying the builder’s public API. This ensures long-term
   extensibility.

5. By keeping the builder stateless, we avoid locks and shared state, which
   dramatically improves parallelizability and simplifies testing.
*/

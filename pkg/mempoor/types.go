package mempoor

import (
	"errors"
	"time"
)

// NodeConfig holds runtime settings for the node.
type NodeConfig struct {
	ListenAddr    string
	BlockInterval time.Duration
	GasLimit      uint64
	MaxTxPerBlock int
	MinFee        uint64
}

// BlockHeader contains minimal metadata describing a block.
type BlockHeader struct {
	Height    uint64
	PrevHash  [32]byte
	Timestamp time.Time

	TxCount int
	GasUsed uint64
}

// Block wraps a header with its ordered transactions.
type Block struct {
	Header       BlockHeader
	Transactions []*Tx
}

// BlockConstraints defines limits used by the block builder when
// requesting transactions from the mempool.
type BlockConstraints struct {
	GasLimit uint64 // maximum total gas allowed in the block
	MaxTx    int    // maximum number of transactions to include
	MinFee   uint64 // optional minimum fee threshold
}

// BlockSelectionResult represents the set of transactions chosen
// by the mempool for block inclusion.
type BlockSelectionResult struct {
	Transactions []*Tx // ordered by priority
	GasUsed      uint64
}

// TxID uniquely identifies a transaction.
type TxID string

// Tx represents a transaction in the mempool and blocks.
type Tx struct {
	ID        TxID
	Sender    string
	Recipient string
	Fee       uint64
	Gas       uint64
	Payload   string

	// Immutable creation timestamp — part of TxID.
	CreatedAt time.Time

	// Mutable scheduling timestamp — used for priority ordering only.
	Timestamp time.Time
}

// Mempool defines the behavior required by the block builder
// and node runtime. A concrete mempool implementation must be
// concurrency-safe internally.
type Mempool interface {
	// Add inserts a new transaction into the mempool.
	Add(tx *Tx) error

	// Update replaces an existing transaction with the same ID.
	// If the transaction does not exist, the implementation may
	// choose to treat this as an Add or as an error.
	Update(tx *Tx) error

	// Remove deletes a transaction by ID.
	Remove(id TxID) error

	// SelectTransactions atomically selects the highest-priority
	// transactions that satisfy the given constraints.
	//
	// IMPORTANT: This must remove the selected txs from the mempool
	// as part of the same atomic operation.
	SelectTransactions(c BlockConstraints) BlockSelectionResult

	// List returns all transactions currently in the mempool in no
	// particular order. Primarily for CLI and debugging.
	List() []*Tx
}

// ErrEmptyBlock is returned when the mempool provides no transactions
// that satisfy the BlockConstraints. The node daemon decides whether
// to skip block production for this tick.
var ErrEmptyBlock = errors.New("blockbuilder: no transactions selected")

// BlockBuilderConfig specifies the rules used to build blocks.
type BlockBuilderConfig struct {
	GasLimit      uint64
	MaxTxPerBlock int
	MinFee        uint64
}

// BlockBuilder assembles blocks using a mempool and static config.
// It is pure and stateless: the caller supplies prevHash, height, and timestamp.
type BlockBuilder struct {
	mp  Mempool
	cfg BlockBuilderConfig
}

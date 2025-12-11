package mempoor

import (
	"testing"
	"time"
)

// ---- Fake mempool implementation for testing ----

type fakeMempool struct {
	result BlockSelectionResult
}

func (f *fakeMempool) Add(tx *Tx) error     { return nil }
func (f *fakeMempool) Update(tx *Tx) error  { return nil }
func (f *fakeMempool) Remove(id TxID) error { return nil }
func (f *fakeMempool) List() []*Tx          { return nil }
func (f *fakeMempool) SelectTransactions(c BlockConstraints) BlockSelectionResult {
	return f.result
}

// ---- Tests ----

// Ensure builder returns ErrEmptyBlock when mempool yields zero txs.
func TestBuildBlock_EmptySelection(t *testing.T) {
	mp := &fakeMempool{
		result: BlockSelectionResult{
			Transactions: nil,
			GasUsed:      0,
		},
	}

	builder := NewBlockBuilder(mp, BlockBuilderConfig{
		GasLimit:      1_000_000,
		MaxTxPerBlock: 100,
		MinFee:        0,
	})

	prev := [32]byte{1, 2, 3}
	height := uint64(10)
	now := time.Now().UTC()

	blk, err := builder.BuildBlock(prev, height, now)
	if err != ErrEmptyBlock {
		t.Fatalf("expected ErrEmptyBlock, got blk=%+v err=%v", blk, err)
	}
}

// Ensure builder propagates header fields and selected txs correctly.
func TestBuildBlock_HeaderAndTxs(t *testing.T) {
	tx1 := &Tx{
		ID:        "tx1",
		Sender:    "alice",
		Recipient: "bob",
		Fee:       10,
		Gas:       50,
		CreatedAt: time.Now().UTC(),
		Timestamp: time.Now().UTC(),
	}

	tx2 := &Tx{
		ID:        "tx2",
		Sender:    "carol",
		Recipient: "dave",
		Fee:       20,
		Gas:       30,
		CreatedAt: time.Now().UTC(),
		Timestamp: time.Now().UTC(),
	}

	mp := &fakeMempool{
		result: BlockSelectionResult{
			Transactions: []*Tx{tx1, tx2},
			GasUsed:      80,
		},
	}

	cfg := BlockBuilderConfig{
		GasLimit:      1_000_000,
		MaxTxPerBlock: 100,
		MinFee:        0,
	}
	builder := NewBlockBuilder(mp, cfg)

	prev := [32]byte{9, 9, 9}
	height := uint64(7)
	now := time.Unix(12345, 0).UTC()

	blk, err := builder.BuildBlock(prev, height, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Header fields
	if blk.Header.Height != height {
		t.Fatalf("expected height=%d, got %d", height, blk.Header.Height)
	}
	if blk.Header.PrevHash != prev {
		t.Fatalf("prevHash mismatch")
	}
	if !blk.Header.Timestamp.Equal(now) {
		t.Fatalf("timestamp mismatch; expected %v got %v", now, blk.Header.Timestamp)
	}
	if blk.Header.TxCount != 2 {
		t.Fatalf("expected TxCount=2, got %d", blk.Header.TxCount)
	}
	if blk.Header.GasUsed != 80 {
		t.Fatalf("expected GasUsed=80, got %d", blk.Header.GasUsed)
	}

	// Transactions should be returned as-is
	if len(blk.Transactions) != 2 {
		t.Fatalf("expected 2 txs, got %d", len(blk.Transactions))
	}
	if blk.Transactions[0].ID != "tx1" || blk.Transactions[1].ID != "tx2" {
		t.Fatalf("unexpected tx order: %+v", blk.Transactions)
	}
}

// Ensure builder is stateless: repeated calls produce independent results.
func TestBuildBlock_Statelessness(t *testing.T) {
	tx := &Tx{
		ID:        "txX",
		Sender:    "alice",
		Recipient: "bob",
		Fee:       1,
		Gas:       10,
		CreatedAt: time.Now().UTC(),
		Timestamp: time.Now().UTC(),
	}

	mp := &fakeMempool{
		result: BlockSelectionResult{
			Transactions: []*Tx{tx},
			GasUsed:      10,
		},
	}

	builder := NewBlockBuilder(mp, BlockBuilderConfig{
		GasLimit:      1_000_000,
		MaxTxPerBlock: 10,
		MinFee:        0,
	})

	prev1 := [32]byte{1}
	prev2 := [32]byte{2}

	blk1, _ := builder.BuildBlock(prev1, 1, time.Unix(111, 0).UTC())
	blk2, _ := builder.BuildBlock(prev2, 2, time.Unix(222, 0).UTC())

	if blk1.Header.Height != 1 || blk2.Header.Height != 2 {
		t.Fatalf("builder must not retain height between calls")
	}
	if blk1.Header.PrevHash == blk2.Header.PrevHash {
		t.Fatalf("builder must not retain prevHash between calls")
	}
	if blk1.Header.Timestamp.Equal(blk2.Header.Timestamp) {
		t.Fatalf("builder must not retain timestamps between calls")
	}
}

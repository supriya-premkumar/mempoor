package mempoor

import (
	"testing"
	"time"
)

func newDummyTx(id string) *Tx {
	return &Tx{
		ID:        TxID(id),
		Sender:    "alice",
		Recipient: "bob",
		Fee:       1,
		Gas:       10,
		Payload:   "data",
		Timestamp: time.Now(),
	}
}

func TestBlockHeaderFields(t *testing.T) {
	h := BlockHeader{
		Height:    5,
		PrevHash:  [32]byte{1, 2, 3},
		Timestamp: time.Now(),
		TxCount:   2,
		GasUsed:   500,
	}

	if h.Height != 5 {
		t.Fatalf("expected height=5, got %d", h.Height)
	}
	if h.PrevHash[0] != 1 || h.PrevHash[1] != 2 || h.PrevHash[2] != 3 {
		t.Fatalf("prevHash not set correctly: %+v", h.PrevHash)
	}
	if h.TxCount != 2 {
		t.Fatalf("expected txCount=2, got %d", h.TxCount)
	}
	if h.GasUsed != 500 {
		t.Fatalf("expected gasUsed=500, got %d", h.GasUsed)
	}
}

func TestBlockHashDeterministic(t *testing.T) {
	now := time.Unix(123, 0).UTC()

	b1 := &Block{
		Header: BlockHeader{
			Height:    1,
			PrevHash:  [32]byte{9, 9, 9},
			Timestamp: now,
			TxCount:   1,
			GasUsed:   10,
		},
		Transactions: []*Tx{newDummyTx("tx1")},
	}

	b2 := &Block{
		Header: BlockHeader{
			Height:    1,
			PrevHash:  [32]byte{9, 9, 9},
			Timestamp: now,
			TxCount:   1,
			GasUsed:   10,
		},
		Transactions: []*Tx{newDummyTx("tx1")},
	}

	h1 := b1.Hash()
	h2 := b2.Hash()

	if h1 != h2 {
		t.Fatalf("expected deterministic hash; got %x vs %x", h1, h2)
	}
}

func TestBlockHashChangesWhenTxChanges(t *testing.T) {
	now := time.Unix(123, 0).UTC()

	b1 := &Block{
		Header: BlockHeader{
			Height:    1,
			PrevHash:  [32]byte{0},
			Timestamp: now,
			TxCount:   1,
			GasUsed:   10,
		},
		Transactions: []*Tx{newDummyTx("tx1")},
	}

	b2 := &Block{
		Header: BlockHeader{
			Height:    1,
			PrevHash:  [32]byte{0},
			Timestamp: now,
			TxCount:   1,
			GasUsed:   10,
		},
		Transactions: []*Tx{newDummyTx("tx2")}, // tx ID differs
	}

	if b1.Hash() == b2.Hash() {
		t.Fatalf("expected block hash to change when tx IDs change")
	}
}

func TestBlockHashChangesWhenHeaderChanges(t *testing.T) {
	now := time.Unix(123, 0).UTC()

	b1 := &Block{
		Header: BlockHeader{
			Height:    1,
			PrevHash:  [32]byte{0},
			Timestamp: now,
			TxCount:   1,
			GasUsed:   10,
		},
		Transactions: []*Tx{newDummyTx("tx1")},
	}

	b2 := &Block{
		Header: BlockHeader{
			Height:    2, // changed height
			PrevHash:  [32]byte{0},
			Timestamp: now,
			TxCount:   1,
			GasUsed:   10,
		},
		Transactions: []*Tx{newDummyTx("tx1")},
	}

	if b1.Hash() == b2.Hash() {
		t.Fatalf("expected block hash to change when header fields change")
	}
}

func TestBlockHashSensitiveToTxOrdering(t *testing.T) {
	now := time.Unix(123, 0).UTC()

	b1 := &Block{
		Header: BlockHeader{
			Height:    1,
			PrevHash:  [32]byte{0},
			Timestamp: now,
			TxCount:   2,
			GasUsed:   20,
		},
		Transactions: []*Tx{newDummyTx("tx1"), newDummyTx("tx2")},
	}

	b2 := &Block{
		Header: BlockHeader{
			Height:    1,
			PrevHash:  [32]byte{0},
			Timestamp: now,
			TxCount:   2,
			GasUsed:   20,
		},
		Transactions: []*Tx{newDummyTx("tx2"), newDummyTx("tx1")}, // reversed
	}

	if b1.Hash() == b2.Hash() {
		t.Fatalf("expected block hash to change when tx order differs")
	}
}

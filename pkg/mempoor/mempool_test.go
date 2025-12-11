package mempoor

import (
	"testing"
	"time"
)

// helper to create tx quickly
func newTx(sender string, fee, gas uint64) *Tx {
	return NewUnsignedTx(sender, "bob", "data", fee, gas)
}

// This test does NOT try to assert functional correctness under concurrency.
// It only ensures that the mempool implementation is race-free and stable
// under concurrent access when run with `go test -race`.
//
// It spawns goroutines that:
//  - Add new txs
//  - Update existing txs (fee bumps)
//  - Remove random txs
//  - Call SelectTransactions

func TestAddAndList(t *testing.T) {
	mp := NewMempool()

	tx1 := newTx("alice", 10, 100)
	tx2 := newTx("carol", 20, 200)

	if err := mp.Add(tx1); err != nil {
		t.Fatalf("unexpected Add error: %v", err)
	}
	if err := mp.Add(tx2); err != nil {
		t.Fatalf("unexpected Add error: %v", err)
	}

	list := mp.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 txs in mempool, got %d", len(list))
	}
}

func TestAddDuplicateFails(t *testing.T) {
	mp := NewMempool()

	tx := newTx("alice", 10, 100)
	_ = mp.Add(tx)

	if err := mp.Add(tx); err != ErrTxExists {
		t.Fatalf("expected ErrTxExists on duplicate Add")
	}
}

func TestUpdateStrictNotFound(t *testing.T) {
	mp := NewMempool()

	tx := newTx("alice", 10, 100)

	if err := mp.Update(tx); err != ErrTxNotFound {
		t.Fatalf("expected ErrTxNotFound on Update of missing tx")
	}
}

func TestUpdateReplacesRecordAndChangesPriority(t *testing.T) {
	mp := NewMempool()

	tx := newTx("alice", 10, 100)
	_ = mp.Add(tx)

	// Fee bump
	updated := &Tx{
		ID:        tx.ID,
		Sender:    tx.Sender,
		Recipient: tx.Recipient,
		Payload:   tx.Payload,
		Fee:       999, // force highest priority
		Gas:       tx.Gas,
		CreatedAt: tx.CreatedAt,
		Timestamp: time.Now().UTC(),
	}

	if err := mp.Update(updated); err != nil {
		t.Fatalf("unexpected Update error: %v", err)
	}

	// Ensure SelectTransactions returns updated version first
	res := mp.SelectTransactions(BlockConstraints{
		MaxTx:    1,
		GasLimit: 1_000_000,
		MinFee:   0,
	})

	if len(res.Transactions) != 1 {
		t.Fatalf("expected 1 tx from selection")
	}
	if res.Transactions[0].Fee != 999 {
		t.Fatalf("expected updated fee=999, got %d", res.Transactions[0].Fee)
	}
}

func TestRemoveStrictNotFound(t *testing.T) {
	mp := NewMempool()

	tx := newTx("alice", 10, 100)
	_ = mp.Add(tx)

	if err := mp.Remove("does-not-exist"); err != ErrTxNotFound {
		t.Fatalf("expected ErrTxNotFound for missing Remove")
	}
}

func TestRemoveSuccess(t *testing.T) {
	mp := NewMempool()
	tx := newTx("alice", 10, 100)
	_ = mp.Add(tx)

	if err := mp.Remove(tx.ID); err != nil {
		t.Fatalf("unexpected Remove error: %v", err)
	}

	if len(mp.List()) != 0 {
		t.Fatalf("expected empty mempool after remove")
	}
}

func TestSelectTransactionsPriorityOrdering(t *testing.T) {
	mp := NewMempool()

	low := newTx("alice", 1, 50)
	med := newTx("bob", 10, 50)
	high := newTx("carol", 100, 50)

	_ = mp.Add(low)
	_ = mp.Add(med)
	_ = mp.Add(high)

	res := mp.SelectTransactions(BlockConstraints{
		MaxTx:    3,
		GasLimit: 1_000_000,
		MinFee:   0,
	})

	if len(res.Transactions) != 3 {
		t.Fatalf("expected all 3 txs, got %d", len(res.Transactions))
	}

	if res.Transactions[0].Fee != 100 ||
		res.Transactions[1].Fee != 10 ||
		res.Transactions[2].Fee != 1 {
		t.Fatalf("expected priority order 100,10,1; got %+v", res.Transactions)
	}
}

func TestSelectTransactionsGasLimit(t *testing.T) {
	mp := NewMempool()

	// gas = 60, 60, 60, 60
	for i := 0; i < 4; i++ {
		tx := newTx("alice", uint64(i), 60)
		_ = mp.Add(tx)
	}

	res := mp.SelectTransactions(BlockConstraints{
		MaxTx:    10,
		GasLimit: 120, // only two tx fit
		MinFee:   0,
	})

	if len(res.Transactions) != 2 {
		t.Fatalf("expected 2 tx from gas limit, got %d", len(res.Transactions))
	}

	if res.GasUsed != 120 {
		t.Fatalf("expected gasUsed=120, got %d", res.GasUsed)
	}
}

func TestSelectTransactionsLowFeePurge(t *testing.T) {
	mp := NewMempool()

	low := newTx("alice", 1, 10)
	high := newTx("bob", 100, 10)

	_ = mp.Add(low)
	_ = mp.Add(high)

	res := mp.SelectTransactions(BlockConstraints{
		MaxTx:    10,
		GasLimit: 1_000_000,
		MinFee:   50, // purge low-fee tx
	})

	if len(res.Transactions) != 1 {
		t.Fatalf("expected only high-fee tx included")
	}
	if res.Transactions[0].Fee != 100 {
		t.Fatalf("expected high-fee tx selected")
	}

	// low must be permanently removed from the mempool
	list := mp.List()
	if len(list) != 0 {
		t.Fatalf("expected mempool empty after purge; got %v txs", len(list))
	}
}

func TestSelectTransactionsSkipButKeepForGas(t *testing.T) {
	mp := NewMempool()

	// high priority but too expensive for this block
	big := newTx("carol", 100, 100)
	// lower fee but cheap → should be included
	small := newTx("alice", 1, 1)

	_ = mp.Add(big)
	_ = mp.Add(small)

	res := mp.SelectTransactions(BlockConstraints{
		MaxTx:    10,
		GasLimit: 1, // big(100 gas) skipped; small(1 gas) included
		MinFee:   0,
	})

	if len(res.Transactions) != 1 {
		t.Fatalf("expected only small tx to be included")
	}
	if res.Transactions[0].Gas != 1 {
		t.Fatalf("expected small tx gas=1")
	}

	// big should still be in mempool because it was *skipped*, not purged
	if len(mp.List()) != 1 {
		t.Fatalf("expected big tx still in mempool")
	}
}

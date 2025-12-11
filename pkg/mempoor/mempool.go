package mempoor

import (
	"container/heap"
	"errors"
	"sync"
)

// Errors exposed by the mempool implementation.
var (
	ErrTxExists   = errors.New("mempool: tx already exists")
	ErrTxNotFound = errors.New("mempool: tx not found")
)

// txRecord is the heap element wrapping a Tx.
type txRecord struct {
	tx    *Tx
	index int // current index in the heap
}

// txHeap is a max-heap ordered by (Fee DESC, Timestamp ASC, ID ASC).
type txHeap []*txRecord

func (h txHeap) Len() int { return len(h) }

func (h txHeap) Less(i, j int) bool {
	ti := h[i].tx
	tj := h[j].tx

	// 1) Higher fee first
	if ti.Fee != tj.Fee {
		return ti.Fee > tj.Fee
	}

	// 2) Earlier timestamp first
	if !ti.Timestamp.Equal(tj.Timestamp) {
		return ti.Timestamp.Before(tj.Timestamp)
	}

	// 3) Stable ordering by TxID
	return ti.ID < tj.ID
}

func (h txHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *txHeap) Push(x any) {
	n := len(*h)
	rec := x.(*txRecord)
	rec.index = n
	*h = append(*h, rec)
}

func (h *txHeap) Pop() any {
	old := *h
	n := len(old)
	rec := old[n-1]
	*h = old[:n-1]
	rec.index = -1
	return rec
}

// mempool is the concrete implementation of the Mempool interface.
// It is concurrency-safe via an internal RWMutex.
type mempool struct {
	mu    sync.RWMutex
	heap  txHeap
	table map[TxID]*txRecord
}

// NewMempool creates an empty, concurrency-safe mempool instance.
func NewMempool() Mempool {
	mp := &mempool{
		table: make(map[TxID]*txRecord),
		heap:  txHeap{},
	}
	heap.Init(&mp.heap)
	return mp
}

// Add inserts a new transaction into the mempool.
//
// NOTE: This assumes tx has already passed basic validation.
// PERF: For production, you might want to enforce max-size bounds here
// and evict lower-priority transactions when full.
func (m *mempool) Add(tx *Tx) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.table[tx.ID]; exists {
		return ErrTxExists
	}

	rec := &txRecord{tx: tx}
	heap.Push(&m.heap, rec)
	m.table[tx.ID] = rec

	return nil
}

// Update replaces an existing transaction with the same ID.
//
// Semantics (locked from Q1/Q2):
//   - Strict: if tx.ID does not exist → ErrTxNotFound.
//   - PUT, not PATCH: we treat tx as a full replacement.
//   - Only fee is logically supposed to change; callers are expected
//     to preserve immutable fields (ID, Sender, Recipient, Payload, CreatedAt).
//
// PERF: For stricter safety, you could enforce immutability here by
// checking old vs new fields and rejecting illegal changes.
func (m *mempool) Update(tx *Tx) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, ok := m.table[tx.ID]
	if !ok {
		return ErrTxNotFound
	}

	// Full replacement of the Tx pointer.
	rec.tx = tx

	// Re-establish heap ordering after fee / timestamp changes.
	heap.Fix(&m.heap, rec.index)

	return nil
}

// Remove deletes a transaction by ID.
//
// Q3 semantics:
// - Strict: if ID not present → ErrTxNotFound.
func (m *mempool) Remove(id TxID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, ok := m.table[id]
	if !ok {
		return ErrTxNotFound
	}

	// Remove from heap and map.
	heap.Remove(&m.heap, rec.index)
	delete(m.table, id)

	return nil
}

// SelectTransactions atomically selects the highest-priority transactions
// that satisfy the given constraints, and removes them from the mempool.
//
// Q4 semantics:
//   - Any tx with Fee < MinFee is purged permanently.
//     It is removed from both heap and table and NOT returned.
//
// Gas limit semantics:
//   - If GasLimit == 0 → no gas limit enforced.
//   - If including a tx would exceed GasLimit, that tx is skipped for this
//     selection but kept in the mempool.
//
// PERF: The simple "skipped" list below is O(k) reinsertion overhead
// per selection. For very large mempools, you could optimize this by
// structuring buckets or using a more advanced scheduler.
func (m *mempool) SelectTransactions(c BlockConstraints) BlockSelectionResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := BlockSelectionResult{
		Transactions: nil,
		GasUsed:      0,
	}

	if c.MaxTx <= 0 || m.heap.Len() == 0 {
		return result
	}

	var skipped []*txRecord

	for len(result.Transactions) < c.MaxTx && m.heap.Len() > 0 {
		rec := heap.Pop(&m.heap).(*txRecord)
		tx := rec.tx

		// 1) Purge low-fee txs permanently.
		if tx.Fee < c.MinFee {
			delete(m.table, tx.ID)
			continue
		}

		// 2) Enforce gas limit (if any).
		if c.GasLimit > 0 && result.GasUsed+tx.Gas > c.GasLimit {
			// Skip this tx for this block, but keep it in mempool.
			skipped = append(skipped, rec)
			continue
		}

		// 3) Accept the tx.
		result.Transactions = append(result.Transactions, tx)
		result.GasUsed += tx.Gas
		delete(m.table, tx.ID)
	}

	// Reinsert skipped txs back into the heap.
	for _, rec := range skipped {
		heap.Push(&m.heap, rec)
		// map entry is still present for skipped txs.
	}

	return result
}

// List returns all transactions currently in the mempool in no particular order.
// Intended for CLI / debugging, not for block production logic.
//
// PERF: This is O(n) over the map. Fine for dev and moderate sizes.
// For extremely large mempools and frequent listing, consider pagination.
func (m *mempool) List() []*Tx {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]*Tx, 0, len(m.table))
	for _, rec := range m.table {
		out = append(out, rec.tx)
	}
	return out
}

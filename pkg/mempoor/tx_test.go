package mempoor

import (
	"testing"
	"time"
)

func TestNewUnsignedTx_GeneratesID(t *testing.T) {
	tx := NewUnsignedTx("alice", "bob", "hello", 10, 500)

	if tx.ID == "" {
		t.Fatalf("expected non-empty tx ID")
	}

	if tx.Sender != "alice" || tx.Recipient != "bob" {
		t.Fatalf("unexpected sender/recipient: %+v", tx)
	}

	if tx.Fee != 10 || tx.Gas != 500 {
		t.Fatalf("unexpected fee/gas: %+v", tx)
	}

	if tx.CreatedAt.IsZero() {
		t.Fatalf("expected CreatedAt to be set")
	}
	if tx.Timestamp.IsZero() {
		t.Fatalf("expected Timestamp to be set")
	}
}

func TestNewUnsignedTx_UniqueIDs(t *testing.T) {
	tx1 := NewUnsignedTx("alice", "bob", "hello", 10, 500)
	tx2 := NewUnsignedTx("alice", "bob", "hello", 10, 500)

	if tx1.ID == tx2.ID {
		t.Fatalf("expected unique IDs; identical creation timestamps are extremely unlikely, but allowed")
	}
}

func TestNewTxUpdate_PreservesIDAndCreatedAt(t *testing.T) {
	origCreated := time.Now().UTC().Add(-1 * time.Minute)
	id := GenerateTxID("alice", "bob", "msg", origCreated)

	tx := NewTxUpdate(id, "alice", "bob", "msg", 5, 100, origCreated)

	if tx.ID != id {
		t.Fatalf("expected ID to be preserved; got %s", tx.ID)
	}

	if !tx.CreatedAt.Equal(origCreated) {
		t.Fatalf("expected CreatedAt to be preserved")
	}

	if tx.Timestamp.IsZero() {
		t.Fatalf("expected Timestamp to be set on update")
	}
}

func TestGenerateTxID_Deterministic(t *testing.T) {
	created := time.Now().UTC()

	id1 := GenerateTxID("a", "b", "p", created)
	id2 := GenerateTxID("a", "b", "p", created)

	if id1 != id2 {
		t.Fatalf("expected deterministic IDs")
	}
}

func TestGenerateTxID_ChangesWithCreatedAt(t *testing.T) {
	ts1 := time.Now().UTC()
	ts2 := ts1.Add(time.Nanosecond)

	id1 := GenerateTxID("a", "b", "p", ts1)
	id2 := GenerateTxID("a", "b", "p", ts2)

	if id1 == id2 {
		t.Fatalf("expected different IDs for different creation times")
	}
}

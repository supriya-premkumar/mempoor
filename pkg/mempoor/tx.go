package mempoor

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"
)

// NewUnsignedTx constructs a tx for "add" workflows.
// TxID is generated based on immutable fields only.
func NewUnsignedTx(sender, recipient, payload string, fee, gas uint64) *Tx {
	created := time.Now().UTC()

	id := GenerateTxID(sender, recipient, payload, created)

	return &Tx{
		ID:        id,
		Sender:    sender,
		Recipient: recipient,
		Payload:   payload,
		Fee:       fee,
		Gas:       gas,
		CreatedAt: created,
		Timestamp: created, // initial arrival timestamp
	}
}

// NewTxUpdate constructs a tx for update workflows.
// ID must be supplied; CreatedAt is preserved.
// Timestamp is refreshed for scheduling.
func NewTxUpdate(id TxID, sender, recipient, payload string, fee, gas uint64, createdAt time.Time) *Tx {
	return &Tx{
		ID:        id,
		Sender:    sender,
		Recipient: recipient,
		Payload:   payload,
		Fee:       fee,
		Gas:       gas,
		CreatedAt: createdAt,
		Timestamp: time.Now().UTC(),
	}
}

// GenerateTxID creates a deterministic ID from immutable fields.
// Fee, Gas, Timestamp DO NOT participate because they may change.
func GenerateTxID(sender, recipient, payload string, createdAt time.Time) TxID {
	raw := sender +
		"|" + recipient +
		"|" + payload +
		"|" + strconv.FormatInt(createdAt.UnixNano(), 10)

	hash := sha256.Sum256([]byte(raw))
	return TxID(hex.EncodeToString(hash[:]))
}

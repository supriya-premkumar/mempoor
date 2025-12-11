package mempoor

import (
	"crypto/sha256"
	"strconv"
	"time"
)

// Hash computes a deterministic block hash.
// See explanation above.
func (b *Block) Hash() [32]byte {
	h := sha256.New()

	h.Write([]byte(
		"height=" + strconv.FormatUint(b.Header.Height, 10) +
			"|timestamp=" + b.Header.Timestamp.UTC().Format(time.RFC3339Nano) +
			"|txcount=" + strconv.Itoa(b.Header.TxCount) +
			"|gasused=" + strconv.FormatUint(b.Header.GasUsed, 10),
	))

	h.Write(b.Header.PrevHash[:])

	for _, tx := range b.Transactions {
		h.Write([]byte(tx.ID))
	}

	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

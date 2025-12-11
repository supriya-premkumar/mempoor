package mempoor

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestMempoolConcurrentAccess(t *testing.T) {
	mp := NewMempool()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Each goroutine gets its own RNG (thread-safe pattern)
	newRNG := func() *rand.Rand {
		return rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	newRandomTx := func(r *rand.Rand) *Tx {
		fee := uint64(r.Intn(1000))
		gas := uint64(r.Intn(50) + 1)
		return NewUnsignedTx("alice", "bob", "payload", fee, gas)
	}

	// --- Goroutine: Add ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		r := newRNG()
		for {
			select {
			case <-stop:
				return
			default:
				_ = mp.Add(newRandomTx(r))
			}
			time.Sleep(time.Millisecond)
		}
	}()

	// --- Goroutine: Update ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		r := newRNG()
		for {
			select {
			case <-stop:
				return
			default:
				list := mp.List()
				if len(list) == 0 {
					continue
				}
				tx := list[r.Intn(len(list))]

				updated := &Tx{
					ID:        tx.ID,
					Sender:    tx.Sender,
					Recipient: tx.Recipient,
					Payload:   tx.Payload,
					Fee:       tx.Fee + 1,
					Gas:       tx.Gas,
					CreatedAt: tx.CreatedAt,
					Timestamp: time.Now().UTC(),
				}

				_ = mp.Update(updated)
			}
			time.Sleep(time.Millisecond)
		}
	}()

	// --- Goroutine: Remove ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		r := newRNG()
		for {
			select {
			case <-stop:
				return
			default:
				list := mp.List()
				if len(list) == 0 {
					continue
				}
				tx := list[r.Intn(len(list))]
				_ = mp.Remove(tx.ID)
			}
			time.Sleep(time.Millisecond)
		}
	}()

	// --- Goroutine: SelectTransactions ---
	wg.Add(1)
	go func() {
		defer wg.Done()
		r := newRNG()
		for {
			select {
			case <-stop:
				return
			default:
				_ = mp.SelectTransactions(BlockConstraints{
					GasLimit: 1_000_000,
					MaxTx:    20,
					MinFee:   uint64(r.Intn(10)),
				})
			}
			time.Sleep(time.Millisecond)
		}
	}()

	// Run for bounded duration
	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()
}

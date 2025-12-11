package mempoor

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

// rpcRequest is the envelope for all incoming RPC calls.
type rpcRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// rpcResponse is the envelope for all outgoing RPC responses.
type rpcResponse struct {
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// ---- Method-specific param/result DTOs ----

type addTxParams struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Payload   string `json:"payload"`
	Fee       uint64 `json:"fee"`
	Gas       uint64 `json:"gas"`
}

type addTxResult struct {
	TxID string `json:"txID"`
}

type updateTxParams struct {
	ID  string `json:"id"`
	Fee uint64 `json:"fee"`
}

type removeTxParams struct {
	ID string `json:"id"`
}

type okResult struct {
	OK bool `json:"ok"`
}

type blockGetParams struct {
	Height uint64 `json:"height"`
}

type listTxResult struct {
	Transactions []*Tx `json:"transactions"`
}

type blockDTO struct {
	Height    uint64    `json:"height"`
	PrevHash  string    `json:"prevHash"`
	Timestamp time.Time `json:"timestamp"`
	TxCount   int       `json:"txCount"`
	GasUsed   uint64    `json:"gasUsed"`
	Hash      string    `json:"hash"`
	Txs       []*Tx     `json:"transactions"`
}

type listBlocksResult struct {
	Blocks []blockDTO `json:"blocks"`
}

type getBlockResult struct {
	Block blockDTO `json:"block"`
}

// handleRPC is the single HTTP entrypoint for all RPC methods.
// It should be mounted on POST /rpc in run.go.
func (n *Node) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeRPCError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPCError(w, http.StatusBadRequest, "invalid JSON request")
		return
	}

	switch req.Method {
	case "tx.add":
		n.rpcTxAdd(w, req.Params)
	case "tx.update":
		n.rpcTxUpdate(w, req.Params)
	case "tx.remove":
		n.rpcTxRemove(w, req.Params)
	case "tx.list":
		n.rpcTxList(w, req.Params)
	case "block.list":
		n.rpcBlockList(w, req.Params)
	case "block.get":
		n.rpcBlockGet(w, req.Params)
	default:
		writeRPCError(w, http.StatusBadRequest, fmt.Sprintf("unknown method %q", req.Method))
	}
}

// ---- tx.add ----

func (n *Node) rpcTxAdd(w http.ResponseWriter, params json.RawMessage) {
	var p addTxParams
	if err := json.Unmarshal(params, &p); err != nil {
		writeRPCError(w, http.StatusBadRequest, "invalid params for tx.add")
		return
	}

	if p.Sender == "" || p.Recipient == "" {
		writeRPCError(w, http.StatusBadRequest, "sender and recipient are required")
		return
	}

	tx := NewUnsignedTx(p.Sender, p.Recipient, p.Payload, p.Fee, p.Gas)
	if err := n.mempool.Add(tx); err != nil {
		writeRPCError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeRPCResult(w, http.StatusOK, addTxResult{TxID: string(tx.ID)})
}

// ---- tx.update ----

func (n *Node) rpcTxUpdate(w http.ResponseWriter, params json.RawMessage) {
	var p updateTxParams
	if err := json.Unmarshal(params, &p); err != nil {
		writeRPCError(w, http.StatusBadRequest, "invalid params for tx.update")
		return
	}

	if p.ID == "" {
		writeRPCError(w, http.StatusBadRequest, "id is required")
		return
	}

	// Find existing tx in mempool to preserve immutable fields.
	// PERF: This is O(n) over List(); acceptable for this project.
	existing := n.findTxByID(TxID(p.ID))
	if existing == nil {
		writeRPCResult(w, http.StatusOK, rpcResponse{Error: ErrTxNotFound.Error()})
		return
	}

	updated := NewTxUpdate(
		existing.ID,
		existing.Sender,
		existing.Recipient,
		existing.Payload,
		p.Fee,
		existing.Gas,
		existing.CreatedAt,
	)

	if err := n.mempool.Update(updated); err != nil {
		writeRPCError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeRPCResult(w, http.StatusOK, okResult{OK: true})
}

// ---- tx.remove ----

func (n *Node) rpcTxRemove(w http.ResponseWriter, params json.RawMessage) {
	var p removeTxParams
	if err := json.Unmarshal(params, &p); err != nil {
		writeRPCError(w, http.StatusBadRequest, "invalid params for tx.remove")
		return
	}

	if p.ID == "" {
		writeRPCError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := n.mempool.Remove(TxID(p.ID)); err != nil {
		if err == ErrTxNotFound {
			writeRPCResult(w, http.StatusOK, rpcResponse{Error: err.Error()})
			return
		}
		writeRPCError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeRPCResult(w, http.StatusOK, okResult{OK: true})
}

// ---- tx.list ----

func (n *Node) rpcTxList(w http.ResponseWriter, params json.RawMessage) {
	// No params expected; ignore any.
	txs := n.mempool.List()

	// Sort in priority order: Fee DESC, Timestamp ASC, ID ASC.
	sort.Slice(txs, func(i, j int) bool {
		ti := txs[i]
		tj := txs[j]

		if ti.Fee != tj.Fee {
			return ti.Fee > tj.Fee
		}
		if !ti.Timestamp.Equal(tj.Timestamp) {
			return ti.Timestamp.Before(tj.Timestamp)
		}
		return ti.ID < tj.ID
	})

	writeRPCResult(w, http.StatusOK, listTxResult{Transactions: txs})
}

// ---- block.list ----

func (n *Node) rpcBlockList(w http.ResponseWriter, params json.RawMessage) {
	// No params expected; ignore.
	n.blocksMu.RLock()
	defer n.blocksMu.RUnlock()

	dtos := make([]blockDTO, 0, len(n.blocks))
	for _, b := range n.blocks {
		dtos = append(dtos, makeBlockDTO(b))
	}

	writeRPCResult(w, http.StatusOK, listBlocksResult{Blocks: dtos})
}

// ---- block.get ----

func (n *Node) rpcBlockGet(w http.ResponseWriter, params json.RawMessage) {
	var p blockGetParams
	if err := json.Unmarshal(params, &p); err != nil {
		writeRPCError(w, http.StatusBadRequest, "invalid params for block.get")
		return
	}

	n.blocksMu.RLock()
	defer n.blocksMu.RUnlock()

	var found *Block
	for _, b := range n.blocks {
		if b.Header.Height == p.Height {
			found = b
			break
		}
	}

	if found == nil {
		writeRPCResult(w, http.StatusOK, rpcResponse{Error: "block not found"})
		return
	}

	dto := makeBlockDTO(found)
	writeRPCResult(w, http.StatusOK, getBlockResult{Block: dto})
}

// ---- helpers ----

// findTxByID does a linear scan over mempool.List().
// PERF: For large mempools, a Get(id) method on Mempool would be better.
func (n *Node) findTxByID(id TxID) *Tx {
	txs := n.mempool.List()
	for _, tx := range txs {
		if tx.ID == id {
			return tx
		}
	}
	return nil
}

func makeBlockDTO(b *Block) blockDTO {
	hash := b.Hash()
	return blockDTO{
		Height:    b.Header.Height,
		PrevHash:  hex.EncodeToString(b.Header.PrevHash[:]),
		Timestamp: b.Header.Timestamp,
		TxCount:   b.Header.TxCount,
		GasUsed:   b.Header.GasUsed,
		Hash:      hex.EncodeToString(hash[:]),
		Txs:       b.Transactions,
	}
}

func writeRPCResult(w http.ResponseWriter, status int, result any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := rpcResponse{
		Result: result,
		Error:  "",
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func writeRPCError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := rpcResponse{
		Result: nil,
		Error:  msg,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

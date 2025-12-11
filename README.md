# mempoor — A Minimal, Deterministic Mempool + Block Builder Node

**mempoor** is a fast, deterministic, production-inspired **mempool and block builder** written entirely in Go.
a.k.a poor man's mempool implementation

It models the *execution-side* of a modern blockchain:

- Priority mempool  
- Fee bumping  
- Deterministic scheduling  
- Stateless block assembly  

**without** implementing consensus, validator rotation, networking, or persistent global state.

It is built for **clarity**, **debuggability**, and **extendability**

---

## ⭐ Features

- Deterministic **priority mempool** (fee DESC, timestamp ASC)
- Strict replace-by-fee semantics (full PUT internally, PATCH-like UX)
- Stateless, pure **block builder**
- In-memory block history (no consensus)
- Simple & extensible **RPC API** (single endpoint)
- Developer-friendly CLI
- Fully concurrency-safe (`go test -race`)
- Clean separation of concerns

---

## 📐 Architecture Overview

```
                 ┌───────────────────────────────┐
                 │             CLI               │
                 │         (via RPC API)         │
                 └───────────────┬───────────────┘
                                 │ POST /rpc
                 ┌───────────────▼────────────────┐
                 │          Node Daemon           │
                 │  - RPC handler                 │
                 │  - Block builder loop          │
                 │  - Mempool + block history     │
                 └───────────────┬────────────────┘
                                 │
         ┌───────────────────────┼────────────────────────┐
         │                       │                        │
┌────────▼────────┐    ┌─────────▼─────────┐     (future executor,
│     Mempool     │    │   BlockBuilder     │      state machine,
│ - max heap PQ   │    │ - stateless        │      consensus engine)
│ - RWMutex       │    │ - deterministic    │
└─────────────────┘    └────────────────────┘
```

---

## 🧱 Core Concepts

### Transactions
- Immutable: `Sender`, `Recipient`, `Payload`, `CreatedAt`
- Mutable: `Fee`, `Timestamp`
- `TxID` derived from immutable fields only

### Mempool
- Max-heap priority queue  
- Strict add/update/remove  
- Low-fee permanent purge  
- Gas-aware selection  
- Internal concurrency safety

### Block Builder
- Pure function: `BuildBlock(prevHash, height, timestamp)`
- Produces block only when ≥1 tx is selected  
- No empty blocks  
- Node controls height + prevHash

### Node Runtime
- Runs block-loop via ticker  
- Stores blocks in-memory  
- Runs RPC server concurrently  
- Clean shutdown via context

---

## 🖥 RPC API (Single Endpoint)

All operations occur via:

```
POST /rpc
```

### Request Example

```json
{
  "method": "tx.add",
  "params": {
    "sender": "alice",
    "recipient": "bob",
    "payload": "hello",
    "fee": 10,
    "gas": 500
  }
}
```

### Successful Response

```json
{
  "result": { "txID": "abc123" },
  "error": null
}
```

### Error Response

```json
{
  "result": null,
  "error": "mempool: tx not found"
}
```

---

## 🔌 Supported RPC Methods

### `tx.add`
Adds a new transaction.

Params:
```json
{
  "sender": "alice",
  "recipient": "bob",
  "payload": "hello",
  "fee": 10,
  "gas": 500
}
```

Response:
```json
{ "txID": "..." }
```

---

### `tx.update`
Fee bump.

Params:
```json
{
  "id": "abc123",
  "fee": 200
}
```

Response:
```json
{ "ok": true }
```

---

### `tx.remove`
Params:
```json
{ "id": "abc123" }
```

Response:
```json
{ "ok": true }
```

---

### `tx.list`
Returns all mempool transactions in priority order.

---

### `block.list`
Returns all blocks produced so far.

### `block.get`
Params:
```json
{ "height": 5 }
```

---

## 🔧 CLI Usage

Start node:
```
mempoor node start --listen localhost:8080
```

Add tx:
```
mempoor tx add \
  --sender alice --recipient bob \
  --payload "hello" --fee 10 --gas 500
```

Update tx:
```
mempoor tx update --id <txID> --fee 200
```

Remove tx:
```
mempoor tx remove --id <txID>
```

List mempool:
```
mempoor tx list
```

List blocks:
```
mempoor block list
```

Get block:
```
mempoor block get --height 0
```

---

## 🧪 Testing

Run the full suite:

```
make 

# make test unit tests runs with -race flag
make test

# make. Saves node output in e2e_test_output
make e2e
```

Includes tests for:

- tx hashing + ID stability  
- mempool ordering + eviction + gas constraints  
- concurrency race test  
- block hashing determinism  
- block builder logic  
- node block loop behavior  

---

## 🚀 Roadmap

- RPC: `/readyz`, `/livez`, `node.status`
- Persistent block store (LevelDB / Badger)
- Execution environment (WASM or custom VM)
- State machine + state root
- Signature verification
- Sharded mempool
- Real P2P networking
- Websocket pub/sub

---

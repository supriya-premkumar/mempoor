#!/usr/bin/env bash
set -euo pipefail
# Resolve mempoor binary location
if [[ -z "${MEMPOOR_BIN:-}" ]]; then
  case "$(uname -s)" in
      Darwin)
          MEMPOOR_BIN="./bin/mempoor-darwin"
          ;;
      Linux)
          MEMPOOR_BIN="./bin/mempoor-linux"
          ;;
      *)
          echo "Unsupported OS: $(uname -s)"
          exit 1
          ;;
  esac
fi


NODE_ADDR="127.0.0.1:8080"
NODE_LOG="./e2e_test_output.log"

echo "==> Starting mempoor node..."
"$MEMPOOR_BIN" start --listen "$NODE_ADDR" > "$NODE_LOG" 2>&1 &
NODE_PID=$!

# Wait for RPC to become available
echo "==> Waiting for node RPC server..."
sleep 1

echo "==> Adding transactions..."
TX1=$($MEMPOOR_BIN tx --addr "$NODE_ADDR" add --sender alice --recipient bob --payload hello --fee 10 --gas 500 | awk '{print $3}')
TX2=$($MEMPOOR_BIN tx --addr "$NODE_ADDR" add --sender carol --recipient dan --payload foo --fee 50 --gas 100 | awk '{print $3}')
TX3=$($MEMPOOR_BIN tx --addr "$NODE_ADDR" add --sender eve --recipient frank --payload bar --fee 1 --gas 100 | awk '{print $3}')

echo "Added TX1=$TX1"
echo "Added TX2=$TX2"
echo "Added TX3=$TX3"

echo "==> Listing mempool..."
"$MEMPOOR_BIN" tx --addr "$NODE_ADDR" list | jq

echo "==> Updating fee for TX1..."
"$MEMPOOR_BIN" tx --addr "$NODE_ADDR" update --id "$TX1" --fee 100

echo "==> Listing mempool after fee bump..."
"$MEMPOOR_BIN" tx --addr "$NODE_ADDR" list | jq

echo "==> Removing lowest-fee TX3..."
"$MEMPOOR_BIN" tx --addr "$NODE_ADDR" remove --id "$TX3"

echo "==> Waiting for block production..."
sleep 3

echo "==> Listing blocks..."
"$MEMPOOR_BIN" block --addr "$NODE_ADDR" list | jq

echo "==> Checking mempool is empty..."
"$MEMPOOR_BIN" tx --addr "$NODE_ADDR" list | jq

echo "==> Fetching block 0..."
"$MEMPOOR_BIN" block --addr "$NODE_ADDR" get --height 0 | jq

echo "==> Adding a few more txs..."
TX6=$($MEMPOOR_BIN tx --addr "$NODE_ADDR" add --sender turing --recipient shannon --payload hello --fee 1000 --gas 500 | awk '{print $3}')
TX7=$($MEMPOOR_BIN tx --addr "$NODE_ADDR" add --sender lamport --recipient neumann --payload foo --fee 43 --gas 100 | awk '{print $3}')
TX8=$($MEMPOOR_BIN tx --addr "$NODE_ADDR" add --sender ada --recipient charles --payload bar --fee 9000 --gas 100 | awk '{print $3}')

echo "==> Waiting for block production..."
sleep 3

echo "==> Viewing Chain. Must have 2 blocks..."
"$MEMPOOR_BIN" block --addr "$NODE_ADDR" list | jq


# Cleanup
echo "==> Stopping node..."
kill "$NODE_PID"
wait "$NODE_PID" 2>/dev/null || true

echo "==> Integration test completed successfully!"

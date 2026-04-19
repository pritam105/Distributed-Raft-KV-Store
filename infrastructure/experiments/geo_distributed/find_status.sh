#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "usage: $0 http://node-a:8000 [http://node-b:8000 ...]" >&2
  exit 1
fi

for node in "$@"; do
  echo "--- $node ---"
  if ! curl -sf --max-time 3 "$node/status"; then
    echo "unreachable"
  fi
  echo
done

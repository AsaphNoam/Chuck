#!/bin/sh
# Structural launcher and owning-workflow routing contract.
set -eu
ROOT=$(CDPATH= cd "$(dirname "$0")/.." && pwd)
if [ "${1-}" = --root ]; then ROOT=$2; fi
exec python3 "$(dirname "$0")/check-launcher-contract.py" "$ROOT"

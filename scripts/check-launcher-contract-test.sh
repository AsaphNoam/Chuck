#!/bin/sh
# Negative structural routing tests; prose paraphrases must continue to pass.
set -eu
exec python3 "$(dirname "$0")/check-launcher-contract-test.py"

#!/usr/bin/env bash
set -euo pipefail

echo "=== Building x86_64 Docker image ==="
docker build --platform linux/amd64 -t simd-crypto-bench .

echo ""
echo "=== Running benchmarks (GOEXPERIMENT=simd) ==="
docker run --platform linux/amd64 --rm simd-crypto-bench

echo ""
echo "=== Done ==="
echo "Tip: pipe to 'tee results.txt' and use 'benchstat' for comparison"
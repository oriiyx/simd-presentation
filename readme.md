# SIMD in Go — Cryptographic XOR Benchmark

Demonstrates Go 1.26's `simd/archsimd` package applied to the core operation behind
stream ciphers (AES-CTR, ChaCha20, OTP): XOR of a plaintext buffer with a keystream.

## What's benchmarked

| Implementation       | Strategy                          | Instructions            |
|----------------------|-----------------------------------|-------------------------|
| `XORScalar`          | Byte-by-byte loop                 | `XOR r8, r8`            |
| `XORSimd256`         | 32 bytes/iteration via AVX2       | `VPXOR ymm, ymm, ymm`   |
| `XORSimd256Unrolled` | 128 bytes/iteration (4× unrolled) | 4× `VPXOR ymm` per iter |

## Run (macOS with Docker)

```bash
./run.sh
```

## Expected speedup

On x86_64 (emulated via Docker on Apple Silicon, real numbers will be higher on native):

- **Small payloads (256B):** ~5-10× faster with SIMD
- **Large payloads (1MB+):** ~15-25× faster with SIMD (memory-bandwidth limited)

On native x86 hardware (e.g. Emerald Rapids Xeon), expect even better numbers since
VPXOR has 1-cycle latency and can retire 3 per cycle on ports p0/p1/p5.

## uops.info reference

For the presentation, show the Emerald Rapids measurements for:

- **VPXOR (YMM, YMM, YMM)** — the instruction behind `Uint8x32.Xor()`
- **VAESENC (XMM, XMM, XMM)** — hardware AES round (for context on crypto acceleration)

Filter: select "Emerald Rapids" architecture, then filter by "AES" or "AVX2" extension.

## Relevance to Netis

Functions that does `sha256.Sum256(...)` — SHA-256 internally uses the same
class of bitwise operations (XOR, AND, rotate, shift) that SIMD accelerates. Go's
`crypto/sha256` already uses hardware SHA extensions when available, but understanding the
SIMD primitives shows *why* it's fast.
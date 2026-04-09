# SIMD in Go — XOR Cipher Performance

> 5-minute lightning talk · Intel Xeon Gold 6548N · Go 1.26 `simd/archsimd`

---

## What is SIMD?

**Single Instruction, Multiple Data.**

Normal (scalar) code processes data one piece at a time. SIMD lets the CPU apply the same operation to many pieces of
data simultaneously, in a single instruction.

```
Scalar (normal):              SIMD:
  A₁ ⊕ B₁  → cycle 1          A₁ ⊕ B₁ ┐
  A₂ ⊕ B₂  → cycle 2          A₂ ⊕ B₂ │
  A₃ ⊕ B₃  → cycle 3          A₃ ⊕ B₃ ├→ 1 cycle (all at once)
  A₄ ⊕ B₄  → cycle 4          A₄ ⊕ B₄ ┘
  = 4 cycles                    = 1 cycle
```

Modern CPUs have special wide registers (128-bit, 256-bit, 512-bit) that hold multiple values at once. With a 256-bit
register (AVX2), you can XOR **32 bytes in a single instruction** instead of one byte at a time.

Most languages don't expose this directly. **Go 1.26** introduces the `simd/archsimd` package, giving Go developers
typed access to these instructions for the first time — no CGo, no assembly.

---

## The CPU: Intel Xeon Gold 6548N (Emerald Rapids)

| Spec              | Value                               |
|-------------------|-------------------------------------|
| Base Clock        | 2.8 GHz (2.8 × 10⁹ cycles/sec)      |
| Cores / Threads   | 32 / 64                             |
| AVX-512 FMA Units | **2 per core**                      |
| L1 Cache          | 80 KB per core                      |
| L2 Cache          | 2 MB per core                       |
| L3 Cache          | 60 MB shared                        |
| ISA Extensions    | SSE4.2 · AVX · AVX2 · AVX-512 · AMX |
| Price             | $3,875                              |

---

## The SIMD Multiplier

**Scalar** (1 byte at a time):

- 2.8 × 10⁹ XOR ops/sec × 1 byte = **~2.8 GB/s**

**AVX2 VPXOR** (32 bytes at a time, 3 execution ports):

- 2.8 × 10⁹ cycles/sec × 3 VPXORs/cycle × 32 bytes = **~268 GB/s theoretical**

> ⚠️ These gains don't come for free — you have to reach for them from code.

---

## The Memory Wall

CPU compute is fast. Feeding it data is the bottleneck.

| Level    | Size         | Latency   | Bandwidth   |
|----------|--------------|-----------|-------------|
| L1 Cache | 80 KB/core   | ~1 ns     | ~1-2 TB/s   |
| L2 Cache | 2 MB/core    | ~3-5 ns   | ~500 GB/s   |
| L3 Cache | 60 MB shared | ~10-15 ns | ~200 GB/s   |
| DDR5 RAM | Up to 4 TB   | ~50-80 ns | ~50-80 GB/s |

**Real throughput = min(CPU compute, memory bandwidth)**

---

## VPXOR ymm — Emerald Rapids (uops.info)

| Metric                      | Value                         |
|-----------------------------|-------------------------------|
| Latency (operand → result)  | **1 cycle**                   |
| Latency (memory → result)   | ≤ 6 cycles                    |
| Throughput (computed)       | **0.33 cycles** (3 per cycle) |
| Throughput (measured, loop) | 0.40 cycles                   |
| Execution ports             | **p0, p1, p5** (3 ports)      |
| μops executed               | 2                             |
| Port usage                  | 1×p015 + 1×p23A               |

The 0.33-cycle throughput means the CPU can retire **3 VPXORs per clock cycle**, each processing 32 bytes = **96
bytes/cycle**.

---

## The Code

Three implementations in `xorcipher/xorcipher.go`:

**Scalar** — byte-by-byte loop:

```go
for i := range plaintext {
dst[i] = plaintext[i] ^ keystream[i]
}
```

**SIMD 256** — 32 bytes per iteration:

```go
p := archsimd.LoadUint8x32((*[32]byte)(plaintext[i: i+32]))
k := archsimd.LoadUint8x32((*[32]byte)(keystream[i: i+32]))
r := p.Xor(k)
r.Store((*[32]byte)(dst[i: i+32]))
```

**SIMD 256 Unrolled** — 128 bytes per iteration (4× unrolled), keeping the pipeline saturated.

---

## Benchmark Results

Docker x86_64 emulation on Apple Silicon (native would be faster):

| Payload | Scalar (MB/s) | SIMD 256 (MB/s) | Unrolled (MB/s) | Speedup |
|---------|---------------|-----------------|-----------------|---------|
| 256B    | 1,378         | 17,715          | 19,197          | ~13-14× |
| 4KB     | 1,401         | 16,888          | 22,189          | ~12-16× |
| 64KB    | 1,389         | 15,694          | 20,827          | ~11-15× |
| 1MB     | 1,365         | 14,386          | 19,027          | ~10-14× |
| 16MB    | 1,360         | 14,098          | 19,179          | ~10-14× |

**Peak throughput: 19.5 GB/s** (unrolled @ 16MB)

---

## Key Takeaways

1. **SIMD is real and accessible in Go 1.26** — `simd/archsimd` gives you typed 256-bit vectors with zero CGo
2. **10-16× speedup on XOR cipher with AVX2** — and this is on emulated x86
3. **Memory bandwidth is the true bottleneck** — L1 → L2 → L3 → RAM; compute is cheap, feeding it is not
4. **Same primitives power AES-CTR, ChaCha20, SHA-256** — understanding XOR SIMD = understanding crypto acceleration

---

## Run It Yourself

```bash
./run.sh
```

Requires Docker. Builds an x86_64 image with Go 1.26 and runs `GOEXPERIMENT=simd go test -bench=.`

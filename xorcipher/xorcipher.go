//go:build goexperiment.simd && amd64

// Package xorcipher demonstrates SIMD-accelerated XOR encryption,
// the core operation behind stream ciphers (AES-CTR, ChaCha20, OTP).
//
// This is a presentation-ready example for "SIMD in Golang" talks,
// showing the Go 1.26 simd/archsimd package applied to cryptography.
package xorcipher

import (
	"simd/archsimd"
)

// XORScalar applies a repeating keystream over plaintext, one byte at a time.
// This is how a naive Go implementation would look.
func XORScalar(dst, plaintext, keystream []byte) {
	for i := range plaintext {
		dst[i] = plaintext[i] ^ keystream[i]
	}
}

// XORSimd256 applies a repeating keystream over plaintext using 256-bit (32-byte)
// SIMD vectors. Each VPXOR instruction XORs 32 bytes in a single cycle.
//
// On uops.info (Emerald Rapids), VPXOR ymm has:
//   - Latency: 1 cycle
//   - Throughput: 0.33 cycles (can retire 3 per cycle)
//   - Ports: p0, p1, p5
//
// This means the CPU can XOR 96 bytes per clock cycle with 256-bit vectors.
func XORSimd256(dst, plaintext, keystream []byte) {
	n := len(plaintext)
	i := 0

	// Process 32 bytes at a time using AVX2 VPXOR
	for i+32 <= n {
		p := archsimd.LoadUint8x32((*[32]byte)(plaintext[i : i+32]))
		k := archsimd.LoadUint8x32((*[32]byte)(keystream[i : i+32]))
		r := p.Xor(k)
		r.Store((*[32]byte)(dst[i : i+32]))
		i += 32
	}

	// Handle remaining bytes (< 32) with scalar fallback
	for ; i < n; i++ {
		dst[i] = plaintext[i] ^ keystream[i]
	}
}

// XORSimd256Unrolled processes 128 bytes per iteration (4x unrolled),
// keeping the SIMD pipeline fully saturated.
// This demonstrates how manual unrolling can help the CPU's out-of-order
// engine overlap loads, XORs, and stores.
func XORSimd256Unrolled(dst, plaintext, keystream []byte) {
	n := len(plaintext)
	i := 0

	// Process 128 bytes per iteration (4 × 32-byte vectors)
	for i+128 <= n {
		p0 := archsimd.LoadUint8x32((*[32]byte)(plaintext[i : i+32]))
		p1 := archsimd.LoadUint8x32((*[32]byte)(plaintext[i+32 : i+64]))
		p2 := archsimd.LoadUint8x32((*[32]byte)(plaintext[i+64 : i+96]))
		p3 := archsimd.LoadUint8x32((*[32]byte)(plaintext[i+96 : i+128]))

		k0 := archsimd.LoadUint8x32((*[32]byte)(keystream[i : i+32]))
		k1 := archsimd.LoadUint8x32((*[32]byte)(keystream[i+32 : i+64]))
		k2 := archsimd.LoadUint8x32((*[32]byte)(keystream[i+64 : i+96]))
		k3 := archsimd.LoadUint8x32((*[32]byte)(keystream[i+96 : i+128]))

		r0 := p0.Xor(k0)
		r1 := p1.Xor(k1)
		r2 := p2.Xor(k2)
		r3 := p3.Xor(k3)

		r0.Store((*[32]byte)(dst[i : i+32]))
		r1.Store((*[32]byte)(dst[i+32 : i+64]))
		r2.Store((*[32]byte)(dst[i+64 : i+96]))
		r3.Store((*[32]byte)(dst[i+96 : i+128]))

		i += 128
	}

	// Remaining full 32-byte blocks
	for i+32 <= n {
		p := archsimd.LoadUint8x32((*[32]byte)(plaintext[i : i+32]))
		k := archsimd.LoadUint8x32((*[32]byte)(keystream[i : i+32]))
		r := p.Xor(k)
		r.Store((*[32]byte)(dst[i : i+32]))
		i += 32
	}

	// Scalar tail
	for ; i < n; i++ {
		dst[i] = plaintext[i] ^ keystream[i]
	}
}

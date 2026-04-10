//go:build goexperiment.simd && amd64

package xorcipher_test

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/oriiyx/simd-presentation/xorcipher"
)

// generateRandomBytes creates cryptographically random data for realistic benchmarks.
func generateRandomBytes(n int) []byte {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return buf
}

// BenchmarkXOR runs all three implementations across common crypto payload sizes.
//
// Size reference for your IOSS RTU context:
//   - 256 B   ~ single RTU credential payload
//   - 4 KB    ~ batch of ~15 RTU credentials
//   - 64 KB   ~ medium batch
//   - 1 MB    ~ large batch signing workload
//   - 16 MB   ~ stress test: signing 1000s of credentials
func BenchmarkXOR(b *testing.B) {
	sizes := []int{
		256,              // single credential
		4 * 1024,         // small batch
		64 * 1024,        // medium batch
		1024 * 1024,      // large batch
		16 * 1024 * 1024, // stress test
	}

	for _, size := range sizes {
		plaintext := generateRandomBytes(size)
		keystream := generateRandomBytes(size)
		destination := make([]byte, size)

		label := formatSize(size)

		b.Run(fmt.Sprintf("Scalar/%s", label), func(b *testing.B) {
			b.SetBytes(int64(size))
			for b.Loop() {
				xorcipher.XORScalar(destination, plaintext, keystream)
			}
		})

		b.Run(fmt.Sprintf("SIMD256/%s", label), func(b *testing.B) {
			b.SetBytes(int64(size))
			for b.Loop() {
				xorcipher.XORSimd256(destination, plaintext, keystream)
			}
		})

		b.Run(fmt.Sprintf("SIMD256_Unrolled/%s", label), func(b *testing.B) {
			b.SetBytes(int64(size))
			for b.Loop() {
				xorcipher.XORSimd256Unrolled(destination, plaintext, keystream)
			}
		})
	}
}

func formatSize(n int) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%dMB", n/(1024*1024))
	case n >= 1024:
		return fmt.Sprintf("%dKB", n/1024)
	default:
		return fmt.Sprintf("%dB", n)
	}
}

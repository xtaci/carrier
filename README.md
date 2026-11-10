# sophon
Error control coding library for go

## GF(256)

The `gf256` package provides polynomial-basis GF(2^8) arithmetic for any
irreducible degree-eight binary polynomial. The polynomial includes its x^8
bit; `0x11d` is commonly used by Reed-Solomon codes and `0x11b` by AES.

```go
import "github.com/xtaci/sophon/gf256"

field, err := gf256.NewField(gf256.ReedSolomonPolynomial)
if err != nil {
    panic(err)
}
product := field.Mul(5, 7)
_ = product

src := []byte{1, 2, 3}
dst := make([]byte, len(src))
field.MulSlice(dst, src, 5)    // dst[i] = 5 * src[i]
field.MulSliceXor(dst, src, 7) // dst[i] ^= 7 * src[i]
```

Construct a field once and reuse it: its lookup tables are immutable and may
be shared across goroutines. Addition and subtraction are `gf256.Add` and
`gf256.Sub`; multiplication, division, inverse and nonnegative integer powers
are field methods. Division by zero and inversion of zero panic; `Pow(0, 0)`
returns 1.

Slice operations require a destination at least as long as the source, permit
exact in-place operation, and reject partial overlaps. They leave any remaining
destination bytes unchanged. The caller must synchronize concurrent access to
buffers.

Batch multiplication uses SSSE3 on supported amd64 CPUs or NEON on arm64, and
falls back to portable Go on other CPUs or when built with `purego` or `noasm`.
Race-instrumented builds also use the Go implementation. These table-based
operations are not constant-time and are not suitable for secret data.

Run `go test ./...` and `go test -tags=purego ./...`; benchmarks are available
with `go test ./gf256 -run '^$' -bench BenchmarkMulSlice`.

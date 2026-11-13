# carrier
Error control coding library for go

## GF(256)

The `gf256` package provides polynomial-basis GF(2^8) arithmetic for any
irreducible degree-eight binary polynomial. The polynomial includes its x^8
bit; `0x11d` is commonly used by Reed-Solomon codes and `0x11b` by AES.

```go
import "github.com/xtaci/carrier/gf256"

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

## Convolutional encoding

The `convolutional` package provides a stateful binary feedforward rate 1/n
encoder. Select a constraint length (1–64) and one or more nonzero generator
polynomials, each fitting within that length. Bit 0 of each polynomial selects
the current input bit and bit i selects the input delayed by i steps.

```go
import "github.com/xtaci/carrier/convolutional"

encoder, err := convolutional.NewEncoder(3, []uint64{0o7, 0o5})
if err != nil {
    panic(err)
}
encoded, err := encoder.Encode([]byte{1, 0, 1, 1})
if err != nil {
    panic(err)
}
// encoded is []byte{1, 1, 1, 0, 0, 0, 0, 1}.
_ = encoded
```

Inputs and outputs use one byte per bit (0 or 1); output bits are interleaved
in generator order. Calls continue the same stream, so call `Reset` before a
new independent stream. No termination bits are added automatically: append
constraint length minus one zero input bits to terminate at the zero state.
An encoder must not be used concurrently without synchronization.
